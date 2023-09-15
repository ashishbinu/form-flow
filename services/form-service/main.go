package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strconv"
	"strings"
	"time"

	"form-service/models"

	ginzap "github.com/gin-contrib/zap"
	"github.com/wagslane/go-rabbitmq"
	"go.uber.org/zap"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgtype"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var (
	db *gorm.DB
)

type DBConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	DBName   string
	SSLMode  string
}

func connectToDatabase() (*gorm.DB, error) {
	dbConfig := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s dbname=%s password=%s sslmode=%s", dbConfig.Host, dbConfig.Port, dbConfig.User, dbConfig.DBName, dbConfig.Password, dbConfig.SSLMode)
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

var logger *zap.Logger
var conn *rabbitmq.Conn
var publisher *rabbitmq.Publisher

func main() {
	var err error

	// core := ecszap.NewCore(ecszap.NewDefaultEncoderConfig(), os.Stdout, zap.DebugLevel)
	// logger = zap.New(core, zap.AddCaller()).With(zap.String("service", "form-service"))
	logger, _ = zap.NewDevelopment()
	logger.With(zap.String("service", "form-service"))

	if db, err = connectToDatabase(); err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	logger.Info("Connected to database")

	dbInstance, err := db.DB()
	if err != nil {
		logger.Fatal("Failed to get database instance", zap.Error(err))
	}
	logger.Info("Database instance created")

	defer dbInstance.Close()

	if err = db.AutoMigrate(&models.Form{}, &models.Question{}, &models.Answer{}, &models.Response{}); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database auto migrated", zap.String("table", "form"), zap.String("table", "question"), zap.String("table", "answer"), zap.String("table", "response"))

	conn, err = rabbitmq.NewConn(
		os.Getenv("RABBITMQ_URL"),
		rabbitmq.WithConnectionOptionsLogging,
	)
	if err != nil {
		logger.Fatal("Failed to connect to RabbitMQ", zap.Error(err))
	}
	logger.Info("Connected to RabbitMQ")
	defer conn.Close()

	publisher, err = rabbitmq.NewPublisher(
		conn,
		rabbitmq.WithPublisherOptionsLogging,
		rabbitmq.WithPublisherOptionsExchangeName("events"),
		rabbitmq.WithPublisherOptionsExchangeDeclare,
	)
	if err != nil {
		logger.Fatal("Failed to create rabbitmq publisher", zap.Error(err))
	}
	logger.Info("RabbitMQ publisher created")
	defer publisher.Close()

	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	v1 := r.Group("/api/v1/form")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1.POST("/", role("team"), createForm)
	v1.GET("/:id", role("user", "team"), getFormByID)
	v1.POST("/responses", role("user"), submitFormResponse)
	v1.GET("/responses/:id", role("user", "team"), getFormResponseByID)
	v1.GET("/:id/responses", role("team"), getFormResponsesByFormID)
	// internal endpoint
	r.GET("/:id/responses", getFormResponsesByFormID)
	r.GET("/:id/answers", getTextAnswerForAQuestion)

	if err = r.Run(":80"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
	logger.Info("Server started", zap.String("address", ":80"))
}

func role(roles ...string) gin.HandlerFunc {

	return func(c *gin.Context) {
		userRole := c.Request.Header.Get("X-Role")
		logger.Debug("Checking user role", zap.String("role", userRole), zap.Any("required_roles", roles))

		roleMatched := false
		for _, role := range roles {
			if userRole == role {
				roleMatched = true
				break
			}
		}

		if !roleMatched {
			logger.Warn("Not a recognized role", zap.String("role", userRole))
			c.JSON(http.StatusForbidden, gin.H{"error": "Not a " + strings.Join(roles, ", ")})
			c.Abort()
			return
		}

		c.Next()
	}
}

// TODO: for all database inserts and errors

func createForm(c *gin.Context) {
	logger.Debug("Entering createForm function")
	// TODO: general schema validation

	type QuestionJsonBody struct {
		Type     models.QuestionType `json:"type"`
		Text     string              `json:"text"`
		Options  []string            `json:"options"`
		Required bool                `json:"required"`
	}

	type FormJsonBody struct {
		Title       string             `json:"title"`
		Description string             `json:"description"`
		Questions   []QuestionJsonBody `json:"questions"`
	}

	var formRequest FormJsonBody

	if err := c.BindJSON(&formRequest); err != nil {
		logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := db.Begin()
	teamId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
	if err != nil {
		logger.Error("Failed to parse X-Id", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	form := models.Form{
		Title:       formRequest.Title,
		Description: formRequest.Description,
		TeamID:      uint(teamId),
	}

	if err := tx.Create(&form).Error; err != nil {
		logger.Error("Failed to create form", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var questions []models.Question
	for i, q := range formRequest.Questions {
		optionArray := pgtype.TextArray{}
		optionArray.Set(q.Options)
		questions = append(questions, models.Question{
			FormID:   form.ID,
			Order:    uint(i + 1),
			Type:     q.Type,
			Text:     q.Text,
			Options:  optionArray,
			Required: q.Required,
		})
	}

	if err := tx.Create(&questions).Error; err != nil {
		logger.Error("Failed to create questions", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Questions created", zap.Uint("form_id", form.ID), zap.Int("count", len(questions)), zap.Uint("team_id", uint(teamId)))

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Form created", zap.Uint("form_id", form.ID))

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Form created successfully",
		"form_id": form.ID,
	})
	logger.Debug("Exiting createForm function")
}

func getFormByID(c *gin.Context) {
	logger.Debug("Entering getFormByID function")
	formID := c.Param("id")
	var form models.Form
	if err := db.Preload("Questions").First(&form, formID).Error; err != nil {
		logger.Error("Failed to get form", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}
	logger.Debug("Form retrieved", zap.Uint("form_id", form.ID))

	response := gin.H{
		"id":          form.ID,
		"title":       form.Title,
		"description": form.Description,
		"questions":   []map[string]interface{}{},
	}

	for _, question := range form.Questions {
		questionData := map[string]interface{}{
			"id":   question.ID,
			"type": question.Type,
			"text": question.Text,
		}
		if options := question.Options.Elements; options != nil {
			questionData["options"] = options
		}

		if question.Required {
			questionData["required"] = true
		}

		response["questions"] = append(response["questions"].([]map[string]interface{}), questionData)
	}

	c.JSON(http.StatusOK, response)
	logger.Debug("Exiting getFormByID function")
}

func submitFormResponse(c *gin.Context) {
	logger.Debug("Entering submitFormResponse function")
	// TODO: Here there generic validation which should follow the required in this
	// NOTE: improve this to use already existing types
	var responseJSON struct {
		FormID  uint `json:"form_id"`
		UserID  uint `json:"user_id"`
		Answers []struct {
			QuestionID uint `json:"question_id"`
			Answer     struct {
				Type  string       `json:"type"`
				Value pgtype.JSONB `json:"value"`
			} `json:"answer"`
		} `json:"answers"`
	}

	if err := c.BindJSON(&responseJSON); err != nil {
		logger.Error("Failed to bind request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := db.Begin()

	userId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
	if err != nil {
		logger.Error("Failed to parse X-Id", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("User retrieved", zap.Uint64("user_id", userId))

	response := models.Response{
		FormID:         responseJSON.FormID,
		UserID:         uint(userId),
		SubmissionTime: time.Now(),
	}

	var form models.Form
	if err = tx.Preload("Questions").First(&form, responseJSON.FormID).Error; err != nil {
		logger.Error("Failed to get form", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	if err := tx.Create(&response).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Response created", zap.Uint("response_id", response.ID))

	var answers []models.Answer
	for _, a := range responseJSON.Answers {
		answers = append(answers, models.Answer{
			QuestionID: a.QuestionID,
			ResponseID: response.ID,
			Type:       a.Answer.Type,
			Value:      a.Answer.Value,
		})
	}

	if err := tx.Create(&answers).Error; err != nil {
		logger.Error("Failed to create answers", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Answers created", zap.Uint("response_id", response.ID), zap.Int("count", len(answers)))

	if err := tx.Commit().Error; err != nil {
		logger.Error("Failed to commit transaction", zap.Error(err))
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Response committed to database", zap.Uint("response_id", response.ID))

	type MessageData struct {
		UserID      uint       `json:"user_id"`
		FormID      uint       `json:"form_id"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		QnA         [][]string `json:"qna"`
	}

	type Message struct {
		Event  string      `json:"event"`
		TeamID uint        `json:"team_id"`
		Data   MessageData `json:"data"`
	}

	message := Message{
		Event:  "response-submission",
		TeamID: form.TeamID,
		Data: MessageData{
			UserID:      uint(userId),
			FormID:      form.ID,
			Title:       form.Title,
			Description: form.Description,
		},
	}

	for _, question := range form.Questions {
		didntAnswer := true
		for _, answer := range answers {
			if answer.QuestionID == question.ID {
				var answerValue interface{}
				switch question.Type {
				case models.Radio:
					var valueIdx uint
					answer.Value.AssignTo(&valueIdx)
					answerValue = question.Options.Elements[valueIdx].String
				case models.Checkbox:
					var valueIdxs []uint
					answer.Value.AssignTo(&valueIdxs)
					var value []string
					for _, idx := range valueIdxs {
						value = append(value, question.Options.Elements[idx].String)
					}
					answerValue = value
				case models.Text:
					answer.Value.AssignTo(&answerValue)
				}
				message.Data.QnA = append(message.Data.QnA, []string{question.Text, convertInterfaceToString(answerValue)})
				didntAnswer = false
			}
		}
		if didntAnswer {
			message.Data.QnA = append(message.Data.QnA, []string{question.Text, "N/A"})
		}

	}

	jsonMessage, err := json.Marshal(message)
	if err != nil {
		logger.Error("Failed to marshal message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if err = publisher.Publish(jsonMessage, []string{"events"}, rabbitmq.WithPublishOptionsContentType("application/json"), rabbitmq.WithPublishOptionsExchange("events")); err != nil {
		logger.Error("Failed to publish message", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":      "success",
		"message":     "Response submitted successfully",
		"response_id": response.ID},
	)
	logger.Debug("Exiting submitFormResponse function")
}

func getFormResponseByID(c *gin.Context) {
	logger.Debug("Entering getFormResponseByID function")
	responseID := c.Param("id")
	var response models.Response
	if err := db.Preload("Answers").First(&response, responseID).Error; err != nil {
		logger.Error("Failed to get response", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Response retrieved", zap.Uint("response_id", response.ID))
	// authorisation
	if c.GetHeader("X-Role") == "user" {
		userId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
		if err != nil {
			logger.Error("Failed to parse X-Id", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if response.UserID != uint(userId) {
			logger.Error("You can't access responses not created by you", zap.Uint("userId", uint(userId)), zap.Uint("requiredUserId", response.UserID))
			c.JSON(http.StatusForbidden, gin.H{"error": "You can't access responses not created by you"})
			return
		}
	} else if c.GetHeader("X-Role") == "team" {
		var form models.Form
		if err := db.First(&form, response.FormID).Error; err != nil {
			logger.Error("Failed to get form", zap.Error(err))
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		logger.Debug("Form retrieved", zap.Uint("form_id", form.ID))

		teamId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
		if err != nil {
			logger.Error("Failed to parse X-Id", zap.Error(err))
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if form.TeamID != uint(teamId) {
			logger.Error("You can't access responses to forms not created by your team", zap.Uint("teamId", uint(teamId)), zap.Uint("requiredTeamId", form.TeamID))
			c.JSON(http.StatusForbidden, gin.H{"error": "You can't access responses to forms not created by your team"})
			return
		}
	} else {
		logger.Error("Access denied", zap.String("role", c.GetHeader("X-Role")), zap.Uint("Id", response.UserID))
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var form models.Form
	if err := db.Preload("Questions").First(&form, response.FormID).Error; err != nil {
		logger.Error("Failed to get form", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated form not found"})
		return
	}
	logger.Debug("Form retrieved", zap.Uint("form_id", form.ID))

	responseJSON := map[string]interface{}{
		"id":              response.ID,
		"submission_time": response.SubmissionTime,
		"user_id":         response.UserID,
		"form": map[string]interface{}{
			"id":          form.ID,
			"title":       form.Title,
			"description": form.Description,
			"questions":   []map[string]interface{}{},
		},
	}

	// TODO: bad code O(n^2) improve
	for _, question := range form.Questions {
		var answerValue interface{}
		for _, answer := range response.Answers {
			if answer.QuestionID == question.ID {
				switch question.Type {
				case models.Radio:
					var valueIdx uint
					answer.Value.AssignTo(&valueIdx)
					answerValue = question.Options.Elements[valueIdx].String
				case models.Checkbox:
					var valueIdxs []uint
					answer.Value.AssignTo(&valueIdxs)
					var value []string
					for _, idx := range valueIdxs {
						value = append(value, question.Options.Elements[idx].String)
					}
					answerValue = value
				case models.Text:
					answer.Value.AssignTo(&answerValue)
				}
				break
			}
		}

		questionJSON := map[string]interface{}{
			"id":     question.ID,
			"type":   question.Type,
			"text":   question.Text,
			"answer": answerValue,
		}

		if options := question.Options.Elements; options != nil {
			questionJSON["options"] = options
		}

		responseJSON["form"].(map[string]interface{})["questions"] = append(
			responseJSON["form"].(map[string]interface{})["questions"].([]map[string]interface{}),
			questionJSON,
		)
	}

	c.JSON(http.StatusOK, responseJSON)
	logger.Debug("Exiting getFormResponseByID function")
}

func getFormResponsesByFormID(c *gin.Context) {
	logger.Debug("Entering getFormResponsesByFormID function")
	formID := c.Param("id")

	var response struct {
		FormID      uint   `json:"form_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Questions   []struct {
			ID    uint   `json:"id"`
			Value string `json:"value"`
		} `json:"questions"`
		Responses []struct {
			UserID  uint `json:"user_id"`
			Answers []struct {
				QuestionID uint        `json:"question_id"`
				Value      interface{} `json:"value"`
			} `json:"answers"`
		} `json:"responses"`
	}

	// Retrieve form data
	var form models.Form
	if err := db.First(&form, formID).Error; err != nil {
		logger.Error("Failed to get form", zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Form retrieved", zap.Uint("form_id", form.ID))

	response.FormID = form.ID
	response.Title = form.Title
	response.Description = form.Description

	// Retrieve question data
	var questions []models.Question
	if err := db.Where("form_id = ?", form.ID).Find(&questions).Error; err != nil {
		logger.Error("Failed to get questions", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Questions retrieved", zap.Uint("form_id", form.ID))

	for _, question := range questions {
		response.Questions = append(response.Questions, struct {
			ID    uint   `json:"id"`
			Value string `json:"value"`
		}{
			ID:    question.ID,
			Value: question.Text,
		})
	}

	// Retrieve response data
	var responses []models.Response
	if err := db.Where("form_id = ?", form.ID).Preload("Answers").Find(&responses).Error; err != nil {
		logger.Error("Failed to get responses", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Responses retrieved", zap.Uint("form_id", form.ID))

	for _, resp := range responses {
		response.Responses = append(response.Responses, struct {
			UserID  uint `json:"user_id"`
			Answers []struct {
				QuestionID uint        `json:"question_id"`
				Value      interface{} `json:"value"`
			} `json:"answers"`
		}{
			UserID: resp.UserID,
		})

		for _, answer := range resp.Answers {
			var question models.Question
			for _, q := range questions {
				if answer.QuestionID == q.ID {
					question = q
					break
				}
			}

			var answerValue interface{}
			switch question.Type {
			case models.Radio:
				var valueIdx uint
				answer.Value.AssignTo(&valueIdx)
				answerValue = question.Options.Elements[valueIdx].String
			case models.Checkbox:
				var valueIdxs []uint
				answer.Value.AssignTo(&valueIdxs)
				var value []string
				for _, idx := range valueIdxs {
					value = append(value, question.Options.Elements[idx].String)
				}
				answerValue = value
			case models.Text:
				answer.Value.AssignTo(&answerValue)
			}
			response.Responses[len(response.Responses)-1].Answers = append(response.Responses[len(response.Responses)-1].Answers, struct {
				QuestionID uint        `json:"question_id"`
				Value      interface{} `json:"value"`
			}{
				QuestionID: answer.QuestionID,
				Value:      answerValue,
			})
		}
	}

	c.JSON(http.StatusOK, response)
	logger.Debug("Exiting getFormResponsesByFormID function")
}

func getTextAnswerForAQuestion(c *gin.Context) {
	responseID := c.Query("response_id")
	questionID := c.Query("question_id")
	var answer models.Answer
	if err := db.Where("response_id = ? AND question_id = ?", responseID, questionID).Find(&answer).Error; err != nil {
		logger.Error("Failed to get answer", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if answer.Type != string(models.Text) {
		logger.Warn("Invalid answer type", zap.String("type", answer.Type))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid answer type"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"value": answer.Value,
	})
}

func convertInterfaceToString(value interface{}) string {
	val := reflect.ValueOf(value)
	if val.Kind() == reflect.Array || val.Kind() == reflect.Slice {
		var result []string
		for i := 0; i < val.Len(); i++ {
			result = append(result, fmt.Sprintf("%v", val.Index(i)))
		}
		return strings.Join(result, ", ")
	}
	return fmt.Sprintf("%v", value)
}
