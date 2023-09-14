package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	pluginserver "sms-service/plugin-server"

	"github.com/google/uuid"
	"github.com/twilio/twilio-go"
	api "github.com/twilio/twilio-go/rest/api/v2010"
)

type SmsService struct {
	pluginserver.DefaultPluginBase
	client *twilio.RestClient
}

type Configuration struct {
}

func New() *SmsService {
	return &SmsService{
		DefaultPluginBase: *pluginserver.NewDefaultPluginBase(),
	}
}

func (gsp *SmsService) Get() pluginserver.PluginData {
	name := "Sms notifier"
	description := "Sms notifier on correct data ingestion"
	url := "http://sms-service"
	actions := []string{}
	events := []string{"response-submission"}
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

func (gsp *SmsService) Initialize() error {
	gsp.client = twilio.NewRestClientWithParams(
		twilio.ClientParams{
			Username: os.Getenv("TWILIO_ACCOUNT_SID"),
			Password: os.Getenv("TWILIO_AUTH_TOKEN"),
		},
	)
	return nil
}

func (gsp *SmsService) Close() error {
	return nil
}

func (gsp *SmsService) Configure(data interface{}) error {
	return nil
}

func (gsp *SmsService) Do(actionName string, data map[string]interface{}) (interface{}, error) {
	return nil, nil
}

var plugin *SmsService

func main() {
	plugin = New()
	plugin.On("response-submission", sendSMS)

	server := pluginserver.New(plugin, "http://plugin-manager-service", os.Getenv("RABBITMQ_URL"))
	defer server.Close()
	err := server.Start(":80")
	if err != nil {
		panic(err)
	}
}

func sms(to string, message string) (string, error) {
	params := &api.CreateMessageParams{}
	params.SetFrom(os.Getenv("TWILIO_PHONE_NUMBER"))
	params.SetTo(to)
	params.SetBody(message)

	resp, err := plugin.client.Api.CreateMessage(params)
	if err != nil {
		return "", err
	}
	response, err := json.Marshal(*resp)
	if err != nil {
		return "", err
	}
	fmt.Println("Response: " + string(response))
	return string(response), nil
}

func sendSMS(data interface{}) (interface{}, error) {
	type MessageData struct {
		UserID      uint       `json:"user_id"`
		FormID      uint       `json:"form_id"`
		Title       string     `json:"title"`
		Description string     `json:"description"`
		QnA         [][]string `json:"qna"`
	}

	type Message struct {
		Event  string      `json:"event"`
		TeamID uint        `json:"team_id"`
		Data   MessageData `json:"data"`
	}
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("Error marshaling data to JSON: %s", err)
	}

	var msg Message
	err = json.Unmarshal(dataBytes, &msg)
	if err != nil {
		return nil, fmt.Errorf("Error unmarshaling data to Message: %s", err)
	}

	userID := fmt.Sprintf("%v", msg.Data.UserID)
	resp, err := http.Get("http://auth-service/users/" + userID)
	if err != nil {
		return nil, fmt.Errorf("Error sending GET request: %s", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var userDetails map[string]interface{}
	err = json.Unmarshal(body, &userDetails)
	if err != nil {
		return nil, err
	}
	msgBody := fmt.Sprintf("\nUser Id : %d\nForm Id : %d\nTitle : %s\nDescription : %s", msg.Data.UserID, msg.Data.FormID, msg.Data.Title, msg.Data.Description)
	for _, qna := range msg.Data.QnA {
		msgBody = msgBody + fmt.Sprintf("\nQuestion : %s\nAnswer : %s\n", qna[0], qna[1])
	}
	result, err := sms(userDetails["phone"].(string), msgBody)
	if err != nil {
		return nil, err
	}

	return result, nil
}
