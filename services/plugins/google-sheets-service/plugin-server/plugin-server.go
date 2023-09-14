package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"slices"

	"github.com/gin-gonic/gin"

	rabbitmq "github.com/wagslane/go-rabbitmq"
)

type PluginServer struct {
	engine           *gin.Engine
	plugin           Plugin
	address          string
	manager          string
	msgQ             string
	rabbitMQConn     *rabbitmq.Conn
	rabbitMQConsumer *rabbitmq.Consumer
}

func New(plugin Plugin, managerUrl string, messageQueueUrl string) *PluginServer {
	return &PluginServer{
		engine:  gin.Default(),
		plugin:  plugin,
		address: ":80",
		manager: managerUrl,
		msgQ:    messageQueueUrl,
	}
}

func (ps *PluginServer) Start(address string) error {
	ps.address = address
	if err := ps.registerPluginWithManager(ps.manager); err != nil {
		return fmt.Errorf("failed to register with plugin manager: %v", err)
	}
	if len(ps.plugin.Get().Events) != 0 {
		if err := ps.initRabbitMQ(); err != nil {
			return fmt.Errorf("failed to initialize RabbitMQ connection: %v", err)
		}
	}

	ps.plugin.Initialize()
	ps.engine.POST("/configure", ps.configurePlugin)
	ps.engine.POST("/actions/:action", ps.executeAction)
	ps.engine.GET("/health", ps.healthCheck)

	if err := ps.engine.Run(ps.address); err != nil {
		return fmt.Errorf("failed to start plugin server: %v", err)
	}

	return nil
}

func (ps *PluginServer) registerPluginWithManager(managerURL string) error {
	registrationData, err := json.Marshal(ps.plugin.Get())
	if err != nil {
		return fmt.Errorf("failed to marshal plugin data to JSON: %v", err)
	}

	req, err := http.NewRequest("POST", managerURL+"/register", bytes.NewBuffer(registrationData))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("registration failed with status code %d", resp.StatusCode)
	}

	return nil
}

func (ps *PluginServer) initRabbitMQ() error {
	conn, err := rabbitmq.NewConn(
		ps.msgQ,
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		return err
	}
	ps.rabbitMQConn = conn

	routingKey := ps.plugin.Get().ID.String()

	consumer, err := rabbitmq.NewConsumer(
		conn,
		func(d rabbitmq.Delivery) rabbitmq.Action {
			ps.handleEvent(d.Body)
			return rabbitmq.Ack
		},
		routingKey,
		rabbitmq.WithConsumerOptionsExchangeName("manager"),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
	)
	if err != nil {
		return err
	}
	ps.rabbitMQConsumer = consumer

	return nil
}

func (ps *PluginServer) handleEvent(eventRawData []byte) {
	var eventData map[string]interface{}
	if err := json.Unmarshal(eventRawData, &eventData); err != nil {
		fmt.Printf("Error decoding event data: %v\n", err)
		return
	}

	eventName, ok := eventData["event"].(string)
	if !ok {
		fmt.Println("Invalid event data format")
		return
	}

	ps.plugin.(*DefaultPluginBase).eventHandlers[eventName](eventData)
}

func (ps *PluginServer) Close() error {
	ps.plugin.Close()
	ps.rabbitMQConsumer.Close()
	if err := ps.rabbitMQConn.Close(); err != nil {
		return err
	}
	return nil
}

func (ps *PluginServer) configurePlugin(c *gin.Context) {
	var configData map[string]interface{}

	if err := c.BindJSON(&configData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}

	if err := ps.plugin.Configure(configData); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Plugin configuration failed",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Plugin configured successfully",
	})
}

func (ps *PluginServer) executeAction(c *gin.Context) {
	actions := ps.plugin.Get().Actions

	if !slices.Contains(actions, c.Param("action")) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid action",
			"actions": actions,
		})
		return
	}

	var actionData map[string]interface{}
	if err := c.BindJSON(&actionData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}

	actionResult, err := ps.plugin.Do(c.Param("action"), actionData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": c.Param("action") + " executed successfully",
		"result":  actionResult,
	})
}

func (ps *PluginServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "plugin": ps.plugin.Get()})
}
