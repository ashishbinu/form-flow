package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

type QuestionType string

const (
	Radio    QuestionType = "radio"
	Text     QuestionType = "text"
	Checkbox QuestionType = "checkbox"
)

type Form struct {
	gorm.Model
	Title       string    `json:"title"`
	Description string    `json:"description"`
	TeamID      uuid.UUID `json:"team_id"`
	Questions   []Question
}

type Question struct {
	gorm.Model
	Order    uint           `json:"order"`
	FormID   uint           `json:"form_id"`
	Type     QuestionType   `json:"type"`
	Text     string         `json:"text"`
	Options  pq.StringArray `json:"options" gorm:"type:text[]"`
	Required bool           `json:"required"`
}

type Response struct {
	gorm.Model
	FormID         uint      `json:"form_id"`
	UserID         uuid.UUID `json:"user_id"`
	SubmissionTime time.Time `json:"submission_time"`
	Form           Form      `gorm:"foreignKey:FormID"`
	Answers        []Answer  // NOTE: you can do some prelaod stuff to it like - `db.Preload("Answers").Find(&form, formID)`
}

type Answer struct {
	gorm.Model
	ResponseID uint        `json:"response_id"`
	QuestionID uint        `json:"question_id"`
	Value      AnswerValue `json:"value" gorm:"type:jsonb"`
	Response   Response    `gorm:"foreignKey:ResponseID"`
	Question   Question    `gorm:"foreignKey:QuestionID"`
}

type AnswerValue struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"` // int, int[], string
}
