package main

import (
	"auth-service/database"
	"auth-service/models"
	"auth-service/utils"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func main() {
	var err error
	r := gin.Default()

	_, err = database.ConnectDB(&database.DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		User:     os.Getenv("DB_USER"),
		Password: os.Getenv("DB_PASSWORD"),
		DBName:   os.Getenv("DB_NAME"),
		SSLMode:  os.Getenv("DB_SSLMODE"),
	})
	if err != nil {
		panic(err)
	}

	// create enum user role
	database.DB.Exec(`DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'user_role') THEN
      CREATE TYPE user_role AS ENUM ('team', 'user');
    END IF;
  END $$;`)
	database.DB.AutoMigrate(&models.User{})

	defer database.CloseDB()

	// NOTE: maybe add verification of email and phone later
	r.POST("/auth/register", Register)
	r.POST("/auth/login", Login)
	r.GET("/auth/validate", Validate)

	r.Run()
}

func Register(c *gin.Context) {

	type RegisterRequest struct {
		Username string          `json:"username"`
		Password string          `json:"password"`
		Email    string          `json:"email"`
		Phone    string          `json:"phone"`
		Role     models.UserRole `json:"role"`
	}

	var registerRequest RegisterRequest

	if err := c.ShouldBindJSON(&registerRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := models.User{
		Username: registerRequest.Username,
		Password: registerRequest.Password,
		Email:    registerRequest.Email,
		Phone:    registerRequest.Phone,
		Role:     registerRequest.Role,
	}

	savedUser, err := user.Register()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "User registered successfully",
		"user":    savedUser,
	})
}

func Login(c *gin.Context) {
	type LoginRequest struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}

	var loginRequest LoginRequest

	if err := c.ShouldBindJSON(&loginRequest); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := models.GetUserByUsername(loginRequest.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	err = user.CheckPassword(loginRequest.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	token, err := utils.GenerateJWT(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"user":    user,
		"token":   token,
	})

}

func Validate(c *gin.Context) {
	claims, err := utils.ValidateJWT(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Token is valid",
		"claims":  claims,
	})
}
