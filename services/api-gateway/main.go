package main

import (
	"fmt"
	"net/http/httputil"
	"net/url"

	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()

	r.Any("/auth/*proxy_path", proxyPass("http://auth-service"))
	r.Any("/form/*proxy_path", proxyPass("http://form-service"))

	r.Run(":80")
}

func proxyPass(target string) gin.HandlerFunc {
	targetUrl, _ := url.Parse(target)
	return func(c *gin.Context) {
		targetUrl.Path = c.Param("proxy_path")
		fmt.Println(targetUrl)

		proxy := httputil.NewSingleHostReverseProxy(targetUrl)
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
