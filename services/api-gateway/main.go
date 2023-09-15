package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/gin-gonic/gin"

	ginzap "github.com/gin-contrib/zap"
	"go.uber.org/zap"
)

var logger *zap.Logger

func main() {
	r := gin.New()
	r.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	r.Use(ginzap.RecoveryWithZap(logger, true))
	logger, _ = zap.NewDevelopment()
	logger.With(zap.String("service", "api-gateway"))

	v1 := r.Group("/api/v1")
	v1.Any("/form/*path", isAuthorised, reverseProxy("http://form-service"))
	v1.Any("/auth/*path", reverseProxy("http://auth-service"))
	v1.Any("/plugins/*path", isAuthorised, reverseProxy("http://plugin-manager-service"))

	r.Run(":80")
}

func isAuthorised(c *gin.Context) {
	logger.Debug("Entering isAuthorised function")
	token := c.GetHeader("Authorization")
	logger.Debug("Authorisation token", zap.String("token", token))

	if token == "" {
		logger.Warn("No auth token specified")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "No auth token specified"})
		c.Abort()
		return
	}

	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://auth-service/api/v1/auth/validate", nil)
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != 200 {
		logger.Warn("Failed to validate token", zap.Error(err), zap.Int("status_code", resp.StatusCode))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Warn("Failed to read response body", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}

	type JSONResponse struct {
		Claims struct {
			ID   int    `json:"id"`
			Role string `json:"role"`
		} `json:"claims"`
		Message string `json:"message"`
	}
	var response JSONResponse
	if err := json.Unmarshal(body, &response); err != nil {
		logger.Warn("Failed to unmarshal response", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		c.Abort()
		return
	}
	logger.Debug("Token validated", zap.Any("response", response))

	id := response.Claims.ID
	role := response.Claims.Role

	c.Request.Header.Set("X-Id", fmt.Sprint(id))
	c.Request.Header.Set("X-Role", role)
	logger.Debug("X-Id and X-Role set", zap.Int("id", id), zap.String("role", role))

	defer resp.Body.Close()
	c.Next()
}

func reverseProxy(target string) gin.HandlerFunc {
	reverseProxyLogger := logger.With(zap.String("target", target))
	targetURL, err := url.Parse(target)

	if err != nil {
		reverseProxyLogger.Fatal("Failed to parse target URL", zap.Error(err))
	}

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", targetURL.Host)
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
		reverseProxyLogger.Debug("Request forwarded to", zap.String("target", target))
	}
}
