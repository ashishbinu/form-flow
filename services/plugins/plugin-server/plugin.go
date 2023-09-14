package server

import (
	"errors"
	"github.com/google/uuid"
)

// EventCallback represents the callback function for handling events.
type EventCallback func(interface{})

// PluginData represents the metadata of the plugin.
type PluginData struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Url         string    `json:"url"`
	Events      []string  `json:"events"`
	Actions     []string  `json:"actions"`
}

// Plugin interface defines the methods a plugin must implement.
type Plugin interface {
	// Get returns the metadata of the plugin.
	Get() PluginData

	// Initialize is called once when the plugin is loaded.
	Initialize() error

	Close() error

	// Configure is called when the plugin needs to be configured.
	Configure(data interface{}) error

	// Do is called to execute a specific action provided by the plugin.
	Do(actionName string, data map[string]interface{}) (interface{}, error)
}

// DefaultPluginBase is a base implementation of the Plugin interface.
type DefaultPluginBase struct {
	// ... other fields common to all plugins
	eventHandlers map[string]EventCallback
}

// NewDefaultPluginBase creates a new instance of DefaultPluginBase.
func NewDefaultPluginBase() *DefaultPluginBase {
	return &DefaultPluginBase{
		eventHandlers: make(map[string]EventCallback),
	}
}

// Get returns the metadata of the plugin.
func (p *DefaultPluginBase) Get() PluginData {
	return PluginData{}
}

// Initialize is called once when the plugin is loaded.
func (p *DefaultPluginBase) Initialize() error {
	return errors.New("Initialize method not implemented")
}

func (p *DefaultPluginBase) Close() error {
	return errors.New("Close method not implemented")
}

// Configure is called when the plugin needs to be configured.
func (p *DefaultPluginBase) Configure(data interface{}) error {
	return errors.New("Configure method not implemented")
}

// Do is called to execute a specific action provided by the plugin.
func (p *DefaultPluginBase) Do(actionName string, data map[string]interface{}) (interface{}, error) {
	return nil, errors.New("Do method not implemented")
}

// On is called to register an event handler for a specific event name.
func (p *DefaultPluginBase) On(eventName string, handler EventCallback) {
	p.eventHandlers[eventName] = handler
}
