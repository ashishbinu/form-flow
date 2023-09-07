package models

import (
	"time"

	"github.com/jackc/pgtype"
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
	Title       string `json:"title"`
	Description string `json:"description"`
	TeamID      uint   `json:"team_id"`
	Questions   []Question
}

type Question struct {
	gorm.Model
	Order    uint             `json:"order"`
	FormID   uint             `json:"form_id"`
	Type     QuestionType     `json:"type"`
	Text     string           `json:"text"`
	Options  pgtype.TextArray `json:"options" gorm:"type:text[]"`
	Required bool             `json:"required"`
}

type Response struct {
	gorm.Model
	FormID         uint      `json:"form_id"`
	UserID         uint      `json:"user_id"`
	SubmissionTime time.Time `json:"submission_time"`
	Form           Form      `gorm:"foreignKey:FormID"`
	Answers        []Answer  // NOTE: you can do some prelaod stuff to it like - `db.Preload("Answers").Find(&form, formID)`
}

type Answer struct {
	gorm.Model
	ResponseID uint         `json:"response_id"`
	QuestionID uint         `json:"question_id"`
	Type       string       `json:"type"`
	Value      pgtype.JSONB `json:"value" gorm:"type:jsonb"` // It can have values of type int, []int, string
	Response   Response     `gorm:"foreignKey:ResponseID"`
	Question   Question     `gorm:"foreignKey:QuestionID"`
}
