package main

import (
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func main() {
	r := gin.Default()

	logger := logrus.New()

	logger.SetLevel(logrus.DebugLevel)

	logFile, err := os.Create("app.log")
	if err != nil {
		logger.Fatal(err)
	}

	logger.SetOutput(logFile)

	r.Use(func(c *gin.Context) {
		logger.WithFields(logrus.Fields{
			"method": c.Request.Method,
			"path":   c.Request.URL.Path,
		}).Info("Request received")

		c.Next()
	})

	r.GET("/log", func(c *gin.Context) {
		logger.Info("Log message received")

		c.JSON(http.StatusOK, gin.H{
			"message": "Log message received",
		})
	})

	if err := r.Run(":8080"); err != nil {
		logger.Fatal(err)
	}
}
