package main

import (
	"os"
	pluginserver "sms-service/plugin-server"

	"github.com/google/uuid"
)

type ValidationService struct {
	pluginserver.DefaultPluginBase
}

type Configuration struct {
}

func New() *ValidationService {
	return &ValidationService{
		DefaultPluginBase: *pluginserver.NewDefaultPluginBase(),
	}
}

func (gsp *ValidationService) Get() pluginserver.PluginData {
	name := "Response Validator"
	description := "Validates response against business rules"
	url := "http://validation-service"
	actions := []string{}
	events := []string{}
	id := uuid.NewSHA1(uuid.Nil, []byte(name))

	return pluginserver.PluginData{
		ID:          id,
		Name:        name,
		Description: description,
		Url:         url,
		Actions:     actions,
		Events:      events,
	}
}

func (gsp *ValidationService) Initialize() error {
	return nil
}

func (gsp *ValidationService) Close() error {
	return nil
}

func (gsp *ValidationService) Configure(data interface{}) error {
	return nil
}

func (gsp *ValidationService) Do(actionName string, data map[string]interface{}) (interface{}, error) {
	return nil, nil
}

var plugin *ValidationService

func main() {
	plugin = New()
	// plugin.On("response-submission", sendSMS)

	server := pluginserver.New(plugin, "http://plugin-manager-service", os.Getenv("RABBITMQ_URL"))
	defer server.Close()
	err := server.Start(":80")
	if err != nil {
		panic(err)
	}
}
