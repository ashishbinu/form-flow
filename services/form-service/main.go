package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgtype"

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

	if db, err = connectToDatabase(); err != nil {
		panic(err)
	}

	dbInstance, err := db.DB()
	if err != nil {
		panic(err)
	}

	defer dbInstance.Close()

	db.AutoMigrate(&models.Form{}, &models.Question{}, &models.Answer{}, &models.Response{})

	r := gin.Default()

	v1 := r.Group("/api/v1/form")

	v1.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	v1.POST("/", role("team"), createForm)
	v1.GET("/:id", role("user", "team"), getFormByID)
	v1.POST("/responses", role("user"), submitFormResponse)
	v1.GET("/responses/:id", role("user", "team"), getFormResponseByID)

	r.Run(":80")
}

func role(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.Request.Header.Get("X-Role")

		// // print headers
		// for k, v := range c.Request.Header {
		//   fmt.Printf("%s: %s\n", k, v)
		// }

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

// TODO: for all database inserts and errors

func createForm(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()}) // TODO: log this
		return
	}

	tx := db.Begin()
	teamId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
	if err != nil {
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
}

func submitFormResponse(c *gin.Context) {
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tx := db.Begin()

	userId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
	if err != nil {
		tx.Rollback()
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	response := models.Response{
		FormID:         responseJSON.FormID,
		UserID:         uint(userId),
		SubmissionTime: time.Now(),
	}

	if err := tx.Create(&response).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

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
		"status":      "success",
		"message":     "Response submitted successfully",
		"response_id": response.ID,
	})
}

func getFormResponseByID(c *gin.Context) {
	responseID := c.Param("id")
	var response models.Response
	if err := db.Preload("Answers").First(&response, responseID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	// authorisation
	if c.GetHeader("X-Role") == "user" {
		userId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if response.UserID != uint(userId) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can't access responses not created by you"})
			return
		}
	} else if c.GetHeader("X-Role") == "team" {
		var form models.Form
		if err := db.First(&form, response.FormID).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		teamId, err := strconv.ParseUint(c.GetHeader("X-Id"), 10, 64)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if form.TeamID != uint(teamId) {
			c.JSON(http.StatusForbidden, gin.H{"error": "You can't access responses to forms not created by your team"})
			return
		}
	} else {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	var form models.Form
	if err := db.Preload("Questions").First(&form, response.FormID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Associated form not found"})
		return
	}

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
					answerValue = question.Options.Elements[valueIdx]
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
}
