package models

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Plugin struct {
	gorm.Model
	ID          uuid.UUID `gorm:"type:uuid" json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	Instances   uint      `json:"instances"`
	Actions     []Action  `json:"actions"`
	Events      []Event   `json:"events"`
}

type Action struct {
	gorm.Model
	PluginID uuid.UUID `gorm:"type:uuid" json:"plugin_id"`
	Name     string    `json:"name"`
	Plugin   Plugin    `gorm:"foreignKey:PluginID"`
}

type Event struct {
	gorm.Model
	PluginID uuid.UUID `gorm:"type:uuid" json:"plugin_id"`
	Name     string    `json:"name"`
	Plugin   Plugin    `gorm:"foreignKey:PluginID"`
}

type PluginSetting struct {
	PluginID uuid.UUID `gorm:"type:uuid" json:"plugin_id"`
	TeamID   uint      `json:"team_id"`
	Enabled  bool      `gorm:"default:false" json:"enabled"`
	Plugin   Plugin    `gorm:"foreignKey:PluginID" json:"-"`
}
