package main

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"form-service/models"

	"github.com/gin-gonic/gin"
	// "github.com/joho/godotenv"
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
		fmt.Println("Failed to connect to database")
		return nil, err
	}

	return db, nil
}

func main() {
	var err error

	// if err = godotenv.Load(); err != nil {
	// 	fmt.Println("Error loading .env file")
	// 	panic(err)
	// }

	for {
		if db, err = connectToDatabase(); err == nil {
			break
		}
		fmt.Println("Failed to connect to the database. Retrying in 3 seconds...")
		time.Sleep(3 * time.Second)
	}

	dbInstance, err := db.DB()
	if err != nil {
		panic(err)
	}

	defer dbInstance.Close()

	db.AutoMigrate(&models.Form{}, &models.Question{}, &models.Answer{}, &models.Response{})

	r := gin.Default()

	r.POST("/forms", createForm)
	r.GET("/forms/:id", getFormByID)
	r.POST("/responses", submitFormResponse)
	r.GET("/responses/:id", getFormResponseByID)

	r.Run()
}

// TODO: for all database inserts and errors

func createForm(c *gin.Context) {
	// TODO: general schema validation

	type QuestionJsonBody struct {
		Type     models.QuestionType `json:"type"`
		Text     string              `json:"text"`
		Options  pq.StringArray      `json:"options"`
		Required bool                `json:"required"`
	}

	type FormJsonBody struct {
		Title       string             `json:"title"`
		Description string             `json:"description"`
		Questions   []QuestionJsonBody `json:"questions"`
	}

	var formRequest FormJsonBody

	if err := c.BindJSON(&formRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // TODO: log this
		return
	}

	tx := db.Begin()

	form := models.Form{
		Title:       formRequest.Title,
		Description: formRequest.Description,
		TeamID:      uuid.New(),
	}

	if err := tx.Create(&form).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var questions []models.Question
	for i, q := range formRequest.Questions {
		questions = append(questions, models.Question{
			FormID:   form.ID,
			Order:    uint(i + 1),
			Type:     q.Type,
			Text:     q.Text,
			Options:  q.Options,
			Required: q.Required,
		})
	}

	if err := tx.Create(&questions).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":  "success",
		"message": "Form created successfully",
		"form_id": form.ID,
	})
}

func getFormByID(c *gin.Context) {
	formID := c.Param("id")
	var form models.Form
	if err := db.Preload("Questions").First(&form, formID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Form not found"})
		return
	}

	response := gin.H{
		"id":          form.ID,
		"title":       form.Title,
		"description": form.Description,
		"questions":   []map[string]interface{}{},
	}

	for _, question := range form.Questions {
		questionData := map[string]interface{}{
			"id":      question.ID,
			"type":    question.Type,
			"text":    question.Text,
			"options": question.Options,
		}

		if question.Required {
			questionData["required"] = true
		}

		response["questions"] = append(response["questions"].([]map[string]interface{}), questionData)
	}

	c.JSON(http.StatusOK, response)
}

func submitFormResponse(c *gin.Context) {
	// TODO: Here there generic validation which should follow the required in this
	// NOTE: improve this to use already existing types
	var responseJSON struct {
		FormID  uint      `json:"form_id"`
		UserID  uuid.UUID `json:"user_id"`
		Answers []struct {
			QuestionID uint `json:"question_id"`
			Value      struct {
				Type  string      `json:"type"`
				Value interface{} `json:"value"`
			} `json:"value"`
		} `json:"answers"`
	}

	if err := c.BindJSON(&responseJSON); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := models.Response{
		FormID:         responseJSON.FormID,
		UserID:         uuid.New(), // NOTE: late add auth and get id
		SubmissionTime: time.Now(),
	}

	for _, answerJSON := range responseJSON.Answers {
		answer := models.Answer{
			QuestionID: answerJSON.QuestionID,
			Value: models.AnswerValue{
				Type:  answerJSON.Value.Type,
				Value: answerJSON.Value.Value,
			},
		}
		response.Answers = append(response.Answers, answer)
	}

	if err := db.Create(&response).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"status":      "success",
		"message":     "Response submitted successfully",
		"response_id": response.ID,
	})
}

func getFormResponseByID(c *gin.Context) {
	responseID := c.Param("id")

	var response models.Response
	if err := db.Preload("Answers").First(&response, responseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Response not found"})
		return
	}

	var form models.Form
	if err := db.Preload("Questions").First(&form, response.FormID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated form not found"})
		return
	}

	responseJSON := map[string]interface{}{
		"id":              response.ID,
		"form_id":         response.FormID,
		"user_id":         response.UserID,
		"submission_time": response.SubmissionTime,
		"form": map[string]interface{}{
			"id":          form.ID,
			"title":       form.Title,
			"description": form.Description,
			"questions":   []map[string]interface{}{},
		},
	}

	for _, question := range form.Questions {
		answerJSON := map[string]interface{}{}
		for _, answer := range response.Answers {
			if answer.QuestionID == question.ID {
				answerJSON = map[string]interface{}{
					"type":  answer.Value.Type,
					"value": answer.Value.Value,
				}
				break
			}
		}

		questionJSON := map[string]interface{}{
			"id":      question.ID,
			"type":    question.Type,
			"text":    question.Text,
			"options": question.Options,
			"answer":  answerJSON,
		}

		responseJSON["form"].(map[string]interface{})["questions"] = append(
			responseJSON["form"].(map[string]interface{})["questions"].([]map[string]interface{}),
			questionJSON,
		)
	}

	c.JSON(http.StatusOK, responseJSON)
}
