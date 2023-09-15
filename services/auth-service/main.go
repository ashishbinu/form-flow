package main

import (
	"auth-service/database"
	"auth-service/models"
	"auth-service/utils"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	ginzap "github.com/gin-contrib/zap"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	var err error

	logger, _ = zap.NewDevelopment()
	logger.With(zap.String("service", "auth-service"))

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

	// create enum user role
	database.DB.Exec(`DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
      CREATE TYPE user_role AS ENUM ('team', 'user');
    END IF;
  END $$;`)
	if err := database.DB.AutoMigrate(&models.User{}); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
	}
	logger.Info("Database auto migrated", zap.String("table", "user"))

	defer database.CloseDB()

	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))

	api := r.Group("/api/v1/auth")

	api.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})
	// NOTE: maybe add verification of email and phone later
	api.POST("/register", Register)
	api.POST("/login", Login)
	api.GET("/validate", Validate)

	// internal endpoint
	r.GET("/users/:id", getUserById)

	r.Run(":80")
}

func Register(c *gin.Context) {
	logger.Debug("Entering Register Function")

	type RegisterRequest struct {
		Username string          `json:"username"`
		Password string          `json:"password"`
		Email    string          `json:"email"`
		Phone    string          `json:"phone"`
		Role     models.UserRole `json:"role"`
	}

	var registerRequest RegisterRequest

	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		logger.Error("Failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("registerRequest", zap.Any("registerRequest", registerRequest))

	user := models.User{
		Username: registerRequest.Username,
		Password: registerRequest.Password,
		Email:    registerRequest.Email,
		Phone:    registerRequest.Phone,
		Role:     registerRequest.Role,
	}

	savedUser, err := user.Register()
	if err != nil {
		logger.Error("Failed to register user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Info("User registered", zap.Any("user", savedUser))

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered successfully",
		"user":    savedUser,
	})
	logger.Debug("Exiting Register Function")
}

func Login(c *gin.Context) {
	logger.Debug("Entering Login Function")
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var loginRequest LoginRequest

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		logger.Error("Failed to bind JSON", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("loginRequest", zap.Any("loginRequest", loginRequest))

	user, err := models.GetUserByUsername(loginRequest.Username)
	if err != nil {
		logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("User retrieved", zap.Any("user", user))

	err = user.CheckPassword(loginRequest.Password)
	if err != nil {
		logger.Error("Failed to check password", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("Password matched", zap.Any("user", user))

	token, err := utils.GenerateJWT(user)
	if err != nil {
		logger.Error("Failed to generate JWT", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("JWT generated", zap.Any("token", token))

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    user,
		"token":   token,
	})
	logger.Debug("Exiting Login Function")
}

func Validate(c *gin.Context) {
	logger.Debug("Entering Validate Function")
	claims, err := utils.ValidateJWT(c)
	if err != nil {
		logger.Error("Failed to validate JWT", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("JWT validated", zap.Any("claims", claims))

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
		"claims":  claims,
	})
	logger.Debug("Exiting Validate Function")
}

func getUserById(c *gin.Context) {
	logger.Debug("Entering getUserById Function")
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		logger.Error("Failed to parse id", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("User id retrieved", zap.Uint("id", id))

	var user models.User
	user, err = models.GetUserById(uint(id))
	if err != nil {
		logger.Error("Failed to get user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	logger.Debug("User retrieved", zap.Any("user", user))

	c.JSON(http.StatusOK, user)
	logger.Debug("Exiting getUserById Function")
}
