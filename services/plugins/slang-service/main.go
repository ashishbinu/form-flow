package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	pluginserver "sms-service/plugin-server"
	"strings"

	"github.com/google/uuid"
)

type SlangService struct {
	pluginserver.DefaultPluginBase
}

type Configuration struct {
}

func New() *SlangService {
	return &SlangService{
		DefaultPluginBase: *pluginserver.NewDefaultPluginBase(),
	}
}

func (gsp *SlangService) Get() pluginserver.PluginData {
	name := "Slang Finder"
	description := "Finds slang for an answer"
	url := "http://slang-service"
	actions := []string{"slang"}
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

func (gsp *SlangService) Initialize() error {
	return nil
}

func (gsp *SlangService) Close() error {
	return nil
}

func (gsp *SlangService) Configure(data interface{}) error {
	return nil
}

func (gsp *SlangService) Do(actionName string, data map[string]interface{}) (interface{}, error) {
	switch actionName {
	case "slang":
		// INFO: json structure of the request
		// {
		// "form_id": ,
		// "response_id": ,
		// "question_id_for_city": ,
		// "question_id_for_slang": ,
		// }

		formId := uint(data["form_id"].(float64))
		responseId := uint(data["response_id"].(float64))
		questionIdForCity := uint(data["question_id_for_city"].(float64))
		questionIdForSlang := uint(data["question_id_for_slang"].(float64))

		// NOTE: can run both GET requests concurrently
		city, err := getAnswerByFormId(formId, responseId, questionIdForCity)
		if err != nil {
			return nil, err
		}
		text, err := getAnswerByFormId(formId, responseId, questionIdForSlang)
		if err != nil {
			return nil, err
		}

		slang, err := convertSlangByCity(text, city)
		if err != nil {
			return nil, err
		}
		return slang, nil

	default:
		return nil, errors.New("Invalid action")
	}
}

var plugin *SlangService

// NOTE:This service can work way better with chatgpt api due to unstructured data
func main() {
	plugin = New()

	server := pluginserver.New(plugin, "http://plugin-manager-service", os.Getenv("RABBITMQ_URL"))
	defer server.Close()
	err := server.Start(":80")
	if err != nil {
		panic(err)
	}
}

func getAnswerByFormId(formId uint, responseId uint, questionId uint) (string, error) {
	strFormId := fmt.Sprintf("%d", formId)
	strResponseId := fmt.Sprintf("%d", responseId)
	strQuestionId := fmt.Sprintf("%d", questionId)
	baseURL := "http://form-service/" + strFormId + "/answers"
	params := url.Values{}
	params.Add("response_id", strResponseId)
	params.Add("question_id", strQuestionId)
	fullURL := baseURL + "?" + params.Encode()

	response, err := http.Get(fullURL)
	if err != nil {
		return "", nil
	}
	defer response.Body.Close()

	body, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	// read the city from the body response { "value": "Bangalore" }
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		return "", err
	}
	return data["value"].(string), nil
}

func convertSlangByCity(text, city string) (string, error) {
	langCode, err := findLanguageCodeByCity(city)
	if err != nil || len(langCode) == 0 {
		return "", err
	}
	slang, err := translate(text, langCode)
	if err != nil {
		return "", err
	}
	return slang, nil
}

func translate(text string, targetLanguageCode string) (string, error) {

	api := "https://google-translate113.p.rapidapi.com/api/v1/translator/text"
	payload := strings.NewReader(fmt.Sprintf("from=auto&to=%s&text=%s", targetLanguageCode, url.PathEscape(text)))

	req, err := http.NewRequest("POST", api, payload)
	if err != nil {
		return "", err
	}

	req.Header.Add("content-type", "application/x-www-form-urlencoded")
	// NOTE: put this inside the env file
	req.Header.Add("X-RapidAPI-Key", "0ab2b978edmsh44bf407b4e7fd06p108c32jsne55cebe17dbb")
	req.Header.Add("X-RapidAPI-Host", "google-translate113.p.rapidapi.com")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}

	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}

  // get the trans value from the body json
  var data map[string]interface{}
  err = json.Unmarshal(body, &data)
  if err != nil {
    return "", err
  }
	return data["trans"].(string), nil
}

func findLanguageCodeByCity(city string) (string, error) {
	// NOTE: This can be done once when plugin is initialised to read it once.
	file, err := os.ReadFile("city-to-lang-code.json")
	if err != nil {
		return "", err
	}

	var cityToLangCode map[string]string
	err = json.Unmarshal(file, &cityToLangCode)
	if err != nil {
		return "", err
	}

	langCode, ok := cityToLangCode[strings.ToLower(city)]
	if !ok {
		return "en", nil
	}
	return langCode, nil

}
