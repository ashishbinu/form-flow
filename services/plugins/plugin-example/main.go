package main

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"slices"
)

type PluginMetadata struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	Events      []string  `json:"events"`
	Actions     []string  `json:"actions"`
}

func pluginDetails() PluginMetadata {
	name := "example-plugin"
	description := "This is a example plugin"
	url := "http://example-plugin"
	events := []string{"event1", "event2"}
	actions := []string{"action1", "action2"}

	return PluginMetadata{
		ID:          uuid.NewSHA1(uuid.Nil, []byte(name)),
		Name:        name,
		Description: description,
		Url:         url,
		Events:      events,
		Actions:     actions,
	}

}

// TODO: create interface for this plugins
func main() {
	r := gin.Default()

	r.POST("/configure", Configure)
	r.POST("/actions/:action", Actions)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, pluginDetails())
	})

	r.Run(":80")
}

func Configure(c *gin.Context) {}
func Actions(c *gin.Context) {
	if !slices.Contains(pluginDetails().Actions, c.Param("action")) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid action",
			"actions": pluginDetails().Actions,
		})
		return
	}
}
