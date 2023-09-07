package main

import (
	// "fmt"
	// "io"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	v1 := r.Group("/api/v1")
	v1.GET("/form/*path", isAuthorised, reverseProxy("http://form-service"))
	v1.GET("/auth/*path", reverseProxy("http://auth-service"))

	r.Run(":80")
}

func isAuthorised(c *gin.Context) {
	token := c.GetHeader("Authorization")
	if token == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	client := &http.Client{}
	req, _ := http.NewRequest("GET", "http://auth-service/api/v1/auth/validate", nil)
	req.Header.Add("Authorization", token)
	resp, err := client.Do(req)

	if err != nil || resp.StatusCode != 200 {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}
	body, _ := io.ReadAll(resp.Body)

	type JSONResponse struct {
		Claims struct {
			ID   int    `json:"id"`
			Role string `json:"role"`
		} `json:"claims"`
		Message string `json:"message"`
	}
	var response JSONResponse
	if err := json.Unmarshal(body, &response); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		c.Abort()
		return
	}

	id := response.Claims.ID
	role := response.Claims.Role

	c.Header("X-Id", fmt.Sprint(id))
	c.Header("X-Role", role)

	defer resp.Body.Close()
	c.Next()
}

func reverseProxy(target string) gin.HandlerFunc {
	targetURL, _ := url.Parse(target)

	proxy := httputil.NewSingleHostReverseProxy(targetURL)
	proxy.Director = func(req *http.Request) {
		req.Header.Set("X-Forwarded-Host", req.Host)
		req.Header.Set("X-Origin-Host", targetURL.Host)
		req.URL.Scheme = targetURL.Scheme
		req.URL.Host = targetURL.Host
	}

	return func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
