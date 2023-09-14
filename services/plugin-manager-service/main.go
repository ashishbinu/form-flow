package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"plugin-manager-service/database"
	"plugin-manager-service/models"
	"strconv"
	"strings"
	"sync"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var conn *rabbitmq.Conn
var logger *zap.Logger
var publisher *rabbitmq.Publisher
var consumer *rabbitmq.Consumer

func main() {
	var err error

	logger, _ = zap.NewDevelopment()
	logger.With(zap.String("service", "plugin-manager-service"))

	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	_, err = database.ConnectDB(&database.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	})
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Connected to database")

	if err = database.DB.AutoMigrate(&models.Plugin{}, &models.Action{}, &models.Event{}, &models.PluginSetting{}); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database auto migrated", zap.String("table", "plugin"), zap.String("table", "action"), zap.String("table", "event"), zap.String("table", "plugin_setting"))

	defer database.CloseDB()

	conn, err = rabbitmq.NewConn(
		os.Getenv("RABBITMQ_URL"),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	logger.Info("Connected to RabbitMQ")
	defer conn.Close()

	consumer, err = rabbitmq.NewConsumer(
		conn,
		routeMessages,
		"router",
		rabbitmq.WithConsumerOptionsRoutingKey("events"),
		rabbitmq.WithConsumerOptionsExchangeName("events"),
		rabbitmq.WithConsumerOptionsExchangeDeclare,
	)
	if err != nil {
		logger.Fatal("Failed to create consumer", zap.Error(err))
	}
	defer consumer.Close()

	publisher, err = rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName("manager"),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		logger.Fatal("Failed to create rabbitmq publisher", zap.Error(err))
	}
	logger.Info("RabbitMQ publisher created")
	defer publisher.Close()

	v1 := r.Group("/api/v1/plugins")

	// TODO: see if everything is implemented as said in plugin architecture

	teamEndpoint := v1.Group("/")
	teamEndpoint.Use(role("team"))

	teamEndpoint.GET("/", GetAllPlugins)
	teamEndpoint.GET("/:id", GetPluginsById)
	// only run below thing when team has enabled that plugin
	// POST endpoint for enabling disabling plugin
	teamEndpoint.GET("/:id/settings", GetPluginSettings)
	teamEndpoint.POST("/:id/status", SetPluginStatus)
	teamEndpoint.POST("/:id/configure", ConfigurePlugin)
	teamEndpoint.POST("/:id/actions/:action", SendActionToPlugin)
	// internal endpoint
	r.POST("/register", RegisterPlugin)

	// send request to /health endpoint to check if plugin is running

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "ok",
		})
	})

	var plugins []models.Plugin
	var urls []string
	if err := database.DB.Find(&plugins).Error; err != nil {
		logger.Fatal("Failed to get plugins", zap.Error(err))
	} else {
		for _, plugin := range plugins {
			urls = append(urls, plugin.Url)
		}

		pollingInterval := 10 * time.Second
		var wg sync.WaitGroup

		for _, url := range urls {
			wg.Add(1)
			go pollEndpoint(url+"/health", pollingInterval, &wg)
		}

		go func() {
			r.Run(":80")
		}()

		wg.Wait()
	}
}

func pollEndpoint(endpointURL string, pollingInterval time.Duration, wg *sync.WaitGroup) {
	defer wg.Done()
	for {
		resp, err := http.Get(endpointURL)
		if err != nil {
			log.Printf("Error polling %s: %v\n", endpointURL, err)
			// TODO: after certain retries reduce the instances count for the plugin
			continue // Continue polling even if there's an error
		}
		defer resp.Body.Close()

		// Read and print the response (you can modify this part for your specific use case)
		// For example, you might want to process the response data differently.
		log.Printf("Response from %s: Status %s\n", endpointURL, resp.Status)

		time.Sleep(pollingInterval)
	}
}

func role(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.Request.Header.Get("X-Role")

		roleMatched := false
		for _, role := range roles {
			if userRole == role {
				roleMatched = true
				break
			}
		}

		if !roleMatched {
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a " + strings.Join(roles, ", ")})
			c.Abort()
			return
		}

		c.Next()
	}
}

func GetAllPlugins(c *gin.Context) {
  logger.Debug("---------------hi-------------")
	var plugins []models.Plugin
	if err := database.DB.Preload("Events").Preload("Actions").Find(&plugins).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	type Response struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Events      []string  `json:"events"`
		Actions     []string  `json:"actions"`
		settings    []models.PluginSetting
	}

	var response []Response
	for _, plugin := range plugins {
		var eventNames []string
		for _, event := range plugin.Events {
			eventNames = append(eventNames, event.Name)
		}
		var actionNames []string
		for _, action := range plugin.Actions {
			actionNames = append(actionNames, action.Name)
		}
		response = append(response, Response{
			ID:          plugin.ID,
			Name:        plugin.Name,
			Description: plugin.Description,
			Events:      eventNames,
			Actions:     actionNames,
		})

	}
	c.JSON(http.StatusOK, response)
}

func GetPluginsById(c *gin.Context) {
	id := c.Param("id")

	var plugin models.Plugin
	if err := database.DB.Preload("Events").First(&plugin, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	type Response struct {
		ID          uuid.UUID `json:"id"`
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Events      []string  `json:"events"`
		Actions     []string  `json:"actions"`
	}

	var response Response
	var events []string
	for _, event := range plugin.Events {
		events = append(events, event.Name)
	}
	var actions []string
	for _, action := range plugin.Actions {
		actions = append(actions, action.Name)
	}
	response = Response{
		ID:          plugin.ID,
		Name:        plugin.Name,
		Description: plugin.Description,
		Events:      events,
		Actions:     actions,
	}

	c.JSON(http.StatusOK, response)
}
func GetPluginSettings(c *gin.Context) {
	id := c.Param("id")
	teamID, err := strconv.ParseUint(c.Request.Header.Get("X-Id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var pluginSetting models.PluginSetting

	if err := database.DB.FirstOrCreate(&pluginSetting, models.PluginSetting{
		PluginID: uuid.MustParse(id),
		TeamID:   uint(teamID),
	}).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, pluginSetting)
}

func SetPluginStatus(c *gin.Context) {
	type Request struct {
		Enabled bool `json:"enabled"`
	}

	var request Request
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	teamID, err := strconv.ParseUint(c.Request.Header.Get("X-Id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	pluginSetting := models.PluginSetting{
		PluginID: uuid.MustParse(c.Param("id")),
		TeamID:   uint(teamID),
		Enabled:  request.Enabled,
	}

	if err := database.DB.Where("plugin_id = ? AND team_id = ?", pluginSetting.PluginID, pluginSetting.TeamID).Save(&pluginSetting).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Plugin status updated",
		"plugin":  pluginSetting})
}

func IsPluginEnabled(db *gorm.DB, pluginID uuid.UUID, teamID uint) (bool, error) {
	var pluginSetting models.PluginSetting

	// Find the PluginSetting record for the given pluginID and teamID
	if err := db.Where("plugin_id = ? AND team_id = ?", pluginID, teamID).First(&pluginSetting).Error; err != nil {
		// Handle the error
		return false, err
	}

	// Return the Enabled field of the PluginSetting record
	return pluginSetting.Enabled, nil
}

func ConfigurePlugin(c *gin.Context) {
	var plugin models.Plugin
	id := c.Param("id")
	teamId, err := strconv.ParseUint(c.Request.Header.Get("X-Id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled, err := IsPluginEnabled(database.DB, uuid.MustParse(id), uint(teamId))
	log.Println("---------------------------")
	log.Println(enabled)
	log.Println("---------------------------")
	if err != nil || !enabled {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Plugin is not enabled", "error": err.Error()})
		return
	}

	if err := database.DB.First(&plugin, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reverseProxy(plugin.Url + "/configure")(c)
}

func SendActionToPlugin(c *gin.Context) {
	id := uuid.MustParse(c.Param("id"))

	teamId, err := strconv.ParseUint(c.Request.Header.Get("X-Id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	enabled, err := IsPluginEnabled(database.DB, id, uint(teamId))
	log.Println("---------------------------")
	log.Println(enabled)
	log.Println("---------------------------")
	if err != nil || !enabled {
		log.Println(err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Plugin is not enabled"})
		return
	}

	var plugin models.Plugin
	if err := database.DB.First(&plugin, id).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	reverseProxy(plugin.Url + "/actions/" + c.Param("action"))(c)

}

func RegisterPlugin(c *gin.Context) {
	var request struct {
		Name        string    `json:"name"`
		Description string    `json:"description"`
		Url         string    `json:"url"`
		Actions     []string  `json:"actions"`
		Events      []string  `json:"events"`
		Id          uuid.UUID `json:"id"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	existingPlugin := models.Plugin{}
	if err := database.DB.Where("id = ?", request.Id).First(&existingPlugin).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"message": "Plugin already registered", "plugin": existingPlugin})
		return
	}

  logger.Debug("-------------------------HERE------------------------------")
	tx := database.DB.Begin()

	plugin := models.Plugin{
		Name:        request.Name,
		Description: request.Description,
		Url:         request.Url,
		ID:          request.Id,
		Instances:   1,
	}
	// if already exists send already registered

	if err := tx.Create(&plugin).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var actions []models.Action
	for _, actionName := range request.Actions {
		actions = append(actions, models.Action{Name: actionName,
			PluginID: plugin.ID})
	}

	if len(actions) != 0 {
		if err := tx.Create(&actions).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	var events []models.Event
	for _, eventName := range request.Events {
		events = append(events, models.Event{Name: eventName,
			PluginID: plugin.ID})
	}

  logger.Debug("Registering Events for plugin : ", zap.String("name", plugin.Name), zap.Any("Events",request.Events ))

	if len(events) != 0 {
		if err := tx.Create(&events).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Plugin registered",
		"plugin":  plugin,
	})
}

func reverseProxy(target string) gin.HandlerFunc {
	targetURL, _ := url.Parse(target)
	log.Println("URL :" + targetURL.String())

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", targetURL.Host)
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
		req.URL.Path = targetURL.Path
	}

	proxy.ModifyResponse = func(res *http.Response) error {
		// Read the response body
		body, err := io.ReadAll(res.Body)
		if err != nil {
			return err
		}

		// Create a new io.ReadCloser for the original response body
		res.Body = io.NopCloser(bytes.NewBuffer(body))

		// Log the response body
		log.Println("Response:", string(body))

		return nil
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}

func routeMessages(d rabbitmq.Delivery) rabbitmq.Action {
	var message struct {
		Event  string      `json:"event"`
		TeamID uint        `json:"team_id"`
		Data   interface{} `json:"data"`
	}
	if err := json.Unmarshal(d.Body, &message); err != nil {
		logger.Error("Failed to unmarshal message", zap.Error(err))
		return rabbitmq.NackDiscard
	}
	logger.Debug("Received message at manager publisher", zap.String("event", message.Event), zap.Uint("team_id", message.TeamID), zap.Any("data", message.Data))

	// First look at the event name and see which plugins have that event name inside the events array
	// then see if those plugins are active or not
	// then see if that team id has enabled that plugin
	var plugins []models.Plugin
	if err := database.DB.
		Where("instances > 0").
		Joins("JOIN events ON plugins.id = events.plugin_id").
		Where("events.name = ?", message.Event).
		Find(&plugins).Error; err != nil {
		logger.Error("Failed to query plugins", zap.Error(err))
		return rabbitmq.NackDiscard
	}

  logger.Info("Found plugins", zap.Int("plugins", len(plugins)))

	var noMatchingPlugin bool
	for _, plugin := range plugins {
		noMatchingPlugin = true
		var pluginSetting models.PluginSetting
		if err := database.DB.Where("plugin_id = ? AND team_id = ?", plugin.ID, message.TeamID).First(&pluginSetting).Error; err != nil {
			logger.Error("Failed to query PluginSetting", zap.Error(err))
			continue
		}

		if pluginSetting.Enabled {
			noMatchingPlugin = false
			logger.Info("Routing message",
				zap.String("event", message.Event),
				zap.Uint("team_id", message.TeamID),
				zap.String("plugin_name", plugin.Name),
			)

			jsonMessage, err := json.Marshal(message)
			if err != nil {
				logger.Error("Failed to marshal message", zap.Error(err))
				return rabbitmq.NackDiscard
			}
			if err = publisher.Publish(jsonMessage, []string{plugin.ID.String()}, rabbitmq.WithPublishOptionsContentType("application/json"), rabbitmq.WithPublishOptionsExchange("manager")); err != nil {
				logger.Error("Failed to publish message", zap.Error(err))
				return rabbitmq.NackDiscard
			}

		}
	}

	if noMatchingPlugin {
		logger.Warn("No matching or enabled plugin found for the message",
			zap.String("event", message.Event),
			zap.Uint("team_id", message.TeamID),
		)
		return rabbitmq.NackDiscard
	}

	return rabbitmq.Ack
}
