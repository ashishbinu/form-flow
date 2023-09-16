package server

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"slices"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	rabbitmq "github.com/wagslane/go-rabbitmq"
)

type PluginServer struct {
	engine           *gin.Engine
	plugin           Plugin
	logger           *zap.Logger
	address          string
	manager          string
	msgQ             string
	rabbitMQConn     *rabbitmq.Conn
	rabbitMQConsumer *rabbitmq.Consumer
}

func New(plugin Plugin, managerUrl string, messageQueueUrl string) *PluginServer {
	logger, _ := zap.NewDevelopment()
	logger.With(zap.String("service", plugin.Get().Name))

	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	return &PluginServer{
		engine:  r,
		plugin:  plugin,
		logger:  logger,
		address: ":80",
		manager: managerUrl,
		msgQ:    messageQueueUrl,
	}
}

func (ps *PluginServer) Start(address string) error {
	ps.address = address
	if err := ps.registerPluginWithManager(ps.manager); err != nil {
		ps.logger.Fatal("Failed to register with plugin manager", zap.Error(err))
	}
	if len(ps.plugin.Get().Events) > 0 {
		if err := ps.initRabbitMQ(); err != nil {
			ps.logger.Fatal("Failed to initialize RabbitMQ connection", zap.Error(err))
		}
	}

	if err := ps.plugin.Initialize(); err != nil {
		ps.logger.Fatal("Failed to initialize plugin", zap.Error(err))
	}

	ps.engine.POST("/configure", ps.configurePlugin)
	ps.engine.POST("/actions/:action", ps.executeAction)
	ps.engine.GET("/health", ps.healthCheck)

	if err := ps.engine.Run(ps.address); err != nil {
		ps.logger.Fatal("Failed to start server", zap.Error(err))
	}

	return nil
}

func (ps *PluginServer) registerPluginWithManager(managerURL string) error {
	// events := []string{}
	//
	//  // TODO: make it work; its not going in
	//  if defaultPlugin, ok := interface{}(ps.plugin).(DefaultPluginBase); ok {
	//    if defaultPlugin.GetEventHandlers() != nil {
	//      for event := range defaultPlugin.GetEventHandlers() {
	//        events = append(events, event)
	//      }
	//    }
	//  }
	// pluginData := PluginData{
	// 	ID:          ps.plugin.Get().ID,
	// 	Name:        ps.plugin.Get().Name,
	// 	Description: ps.plugin.Get().Description,
	// 	Url:         ps.plugin.Get().Url,
	// 	Actions:     ps.plugin.Get().Actions,
	// 	Events:      events,
	// }
	// registrationData, err := json.Marshal(pluginData)

	registrationData, err := json.Marshal(ps.plugin.Get())
	if err != nil {
		ps.logger.Error("Failed to marshal plugin data", zap.Error(err))
	}
	ps.logger.Debug("Plugin data", zap.String("data", string(registrationData)))

	req, err := http.NewRequest("POST", managerURL+"/register", bytes.NewBuffer(registrationData))
	if err != nil {
		ps.logger.Error("Failed to create HTTP request", zap.Error(err))
	}
	ps.logger.Debug("HTTP request", zap.Any("request", req))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		ps.logger.Error("HTTP request failed", zap.Error(err))
		return fmt.Errorf("HTTP request failed: %v", err)
	}
	ps.logger.Debug("HTTP response", zap.Int("status_code", resp.StatusCode))
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ps.logger.Error("Registration failed", zap.Int("status_code", resp.StatusCode))
		return fmt.Errorf("registration failed with status code %d", resp.StatusCode)
	}
	ps.logger.Debug("Registration successful")

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
		ps.plugin.Get().Name,
		rabbitmq.WithConsumerOptionsRoutingKey(routingKey),
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
	ps.logger.Debug("Entering handleEvent Function")
	var eventData map[string]interface{}
	if err := json.Unmarshal(eventRawData, &eventData); err != nil {
		ps.logger.Error("Failed to unmarshal event data", zap.Error(err))
		return
	}

	eventName, ok := eventData["event"].(string)
	if !ok {
		ps.logger.Error("Invalid event data format", zap.Any("event", eventData))
		return
	}

	ps.logger.Debug("Event received", zap.String("event", eventName))
	var message interface{} = eventData
	data, err := ps.plugin.GetEventHandlers()[eventName](message)
	if err != nil {
		ps.logger.Error("Event handler failed", zap.Error(err))
		return
	}
	ps.logger.Debug("Event data", zap.Any("data", data))
	ps.logger.Debug("Exiting handleEvent Function")
}

func (ps *PluginServer) Close() error {
	ps.plugin.Close()
	if ps.rabbitMQConsumer != nil {
		ps.rabbitMQConsumer.Close()
	}
	if ps.rabbitMQConn != nil {
		if err := ps.rabbitMQConn.Close(); err != nil {
			ps.logger.Error("Failed to close RabbitMQ connection", zap.Error(err))
			return err
		}
	}
	return nil
}

func (ps *PluginServer) configurePlugin(c *gin.Context) {
	var configData map[string]interface{}

	if err := c.BindJSON(&configData); err != nil {
		ps.logger.Error("Failed to bind JSON data", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}
	ps.logger.Debug("Plugin configuration", zap.Any("configData", configData))

	if err := ps.plugin.Configure(configData); err != nil {
		ps.logger.Error("Plugin configuration failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Plugin configuration failed",
		})
		return
	}
	ps.logger.Debug("Plugin configuration successful")

	c.JSON(http.StatusOK, gin.H{
		"message": "Plugin configured successfully",
	})
}

func (ps *PluginServer) executeAction(c *gin.Context) {
	ps.logger.Debug("Entering executeAction Function")
	actions := ps.plugin.Get().Actions
	ps.logger.Debug("Available actions", zap.Any("actions", actions))

	if !slices.Contains(actions, c.Param("action")) {
		ps.logger.Error("Invalid action", zap.String("action", c.Param("action")))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid action",
			"actions": actions,
		})
		return
	}
	ps.logger.Debug("Executing action", zap.String("action", c.Param("action")))

	var actionData map[string]interface{}
	if err := c.BindJSON(&actionData); err != nil {
		ps.logger.Error("Failed to bind JSON data", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid JSON data",
		})
		return
	}
	ps.logger.Debug("Action data parsed", zap.Any("actionData", actionData))

	actionResult, err := ps.plugin.Do(c.Param("action"), actionData)
	if err != nil {
		ps.logger.Error("Action failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}
	ps.logger.Debug("Action result", zap.Any("actionResult", actionResult))

	c.JSON(http.StatusOK, gin.H{
		"message": c.Param("action") + " executed successfully",
		"result":  actionResult,
	})
	ps.logger.Debug("Exiting executeAction function")
}

func (ps *PluginServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok", "plugin": ps.plugin.Get()})
}
