package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	pluginserver "google-sheets-service/plugin-server"
	"io"
	"net/http"
	"os"

	"strconv"
	"strings"

	// "google-sheets-service/database"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/drive/v3"
	"google.golang.org/api/option"
	"google.golang.org/api/sheets/v4"

	"github.com/google/uuid"
)

type GoogleSheetsPlugin struct {
	pluginserver.DefaultPluginBase
}

type Configuration struct {
}

func New() *GoogleSheetsPlugin {
	return &GoogleSheetsPlugin{}
}

func (gsp *GoogleSheetsPlugin) Get() pluginserver.PluginData {
	name := "google-sheets-exporter"
	description := "Google Sheets Exporter"
	url := "http://google-sheets-service"
	actions := []string{"export"}
	id := uuid.NewSHA1(uuid.Nil, []byte(name))

	return pluginserver.PluginData{
		ID:          id,
		Name:        name,
		Description: description,
		Url:         url,
		Actions:     actions,
	}
}

func (gsp *GoogleSheetsPlugin) Initialize() error {
	return nil
}

func (gsp *GoogleSheetsPlugin) Close() error {
	return nil
}

func (gsp *GoogleSheetsPlugin) Configure(data interface{}) error {
	return nil
}

func (gsp *GoogleSheetsPlugin) Do(actionName string, data map[string]interface{}) (interface{}, error) {
	switch actionName {
	case "export":
		FormID := strconv.FormatUint(uint64(data["form_id"].(float64)), 10)
		resp, err := http.Get("http://form-service/" + FormID + "/responses")
		if err != nil {
			return nil, fmt.Errorf("Error sending GET request: %s", err)
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}

		var formResponsesData RequestJson
		err = json.Unmarshal(body, &formResponsesData)
		if err != nil {
			return nil, err
		}

		data, err := exportToSpreadsheet(formResponsesData)
		if err != nil {
			return nil, err
		}

		return data, nil
	default:
		return nil, errors.New("Invalid action")
	}
}

func main() {
	var plugin GoogleSheetsPlugin
	server := pluginserver.New(&plugin, "http://plugin-manager-service", "")
	defer server.Close()
	err := server.Start(":80")
	if err != nil {
		panic(err)
	}
}

type RequestJson struct {
	FormID      uint   `json:"form_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Questions   []struct {
		ID    uint   `json:"id"`
		Value string `json:"value"`
	} `json:"questions"`
	Responses []struct {
		UserID  uint `json:"user_id"`
		Answers []struct {
			QuestionID uint        `json:"question_id"`
			Value      interface{} `json:"value"`
		} `json:"answers"`
	} `json:"responses"`
}

func exportToSpreadsheet(data RequestJson) (string, error) {
	credentials, err := os.ReadFile("credentials.json")
	if err != nil {
		return "", err
	}

	// Create a new Google Sheets service with a custom HTTP client
	config, err := google.JWTConfigFromJSON(credentials, drive.DriveScope, sheets.SpreadsheetsScope)
	if err != nil {
		return "", err
	}
	client := config.Client(context.Background())

	svc, err := sheets.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return "", err
	}

	// Create a new Google Drive service with a custom HTTP client
	driveSvc, err := drive.NewService(context.Background(), option.WithHTTPClient(client))
	if err != nil {
		return "", err
	}

	// Create a new Spreadsheet
	spreadsheet := &sheets.Spreadsheet{
		Properties: &sheets.SpreadsheetProperties{
			Title: data.Title,
		},
	}

	res, err := svc.Spreadsheets.Create(spreadsheet).Context(context.Background()).Do()
	if err != nil {
		return "", err
	}
	spreadsheetID := res.SpreadsheetId

	// Share the spreadsheet with anyone as a writer
	permission := &drive.Permission{
		Type: "anyone",
		Role: "writer",
	}

	_, err = driveSvc.Permissions.Create(spreadsheetID, permission).Context(context.Background()).Do()
	if err != nil {
		return "", err
	}

	// Write column headers
	var values [][]interface{}
	headers := make([]interface{}, len(data.Questions))
	for i, question := range data.Questions {
		headers[i] = question.Value
	}
	values = append(values, headers)

	// Write row data
	for _, response := range data.Responses {
		row := make([]interface{}, len(data.Questions))
		for i, question := range data.Questions {
			for _, answer := range response.Answers {
				if answer.QuestionID == question.ID {
					row[i] = convertInterfaceToString(answer.Value)
					break
				}
			}
		}
		values = append(values, row)
	}

	writeRange := "Sheet1!A1:" + string(rune('A'+len(data.Questions)-1)) + strconv.Itoa(len(values))
	valueRange := &sheets.ValueRange{
		Values: values,
	}
	_, err = svc.Spreadsheets.Values.Update(spreadsheetID, writeRange, valueRange).ValueInputOption("RAW").Do()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("https://docs.google.com/spreadsheets/d/%s/edit", spreadsheetID), nil
}

func convertInterfaceToString(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%f", v)
	case []interface{}:
		var strValues = make([]string, len(v))
		for i, elem := range v {
			strValues[i] = fmt.Sprintf("%v", elem)
		}
		return strings.Join(strValues, ",")
	default:
		return fmt.Sprintf("%v", value)
	}
}
