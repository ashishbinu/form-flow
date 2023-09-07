package models

import (
	"auth-service/database"
	"html"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type UserRole string

const (
	TeamUserRole UserRole = "team"
	UserUserRole UserRole = "user"
)

type User struct {
	gorm.Model
	Role     UserRole `gorm:"type:user_role;not null;default:'user'" json:"role"`
	Username string   `gorm:"size:255;not null;unique" json:"username"`
	Email    string   `gorm:"size:255;not null;unique" json:"email"`
	Phone    string   `gorm:"size:20;not null" json:"phone"`
	Password string   `gorm:"size:255;not null" json:"-"`
}

func (user *User) Register() (*User, error) {
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(user.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	user.Password = string(passwordHash)
	user.Username = html.EscapeString(strings.TrimSpace(user.Username))

	err = database.DB.Create(&user).Error
	if err != nil {
		return &User{}, err
	}
	return user, nil
}

func GetUsers(User *[]User) (err error) {
	err = database.DB.Find(User).Error
	if err != nil {
		return err
	}
	return nil
}

func GetUserByUsername(username string) (User, error) {
	var user User
	err := database.DB.Where("username=?", username).Find(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func (user *User) CheckPassword(password string) error {
	return bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
}

func GetUserById(id uint) (User, error) {
	var user User
	err := database.DB.Where("id=?", id).Find(&user).Error
	if err != nil {
		return User{}, err
	}
	return user, nil
}

func GetUser(User *User, id int) (err error) {
	err = database.DB.Where("id = ?", id).First(User).Error
	if err != nil {
		return err
	}
	return nil
}

func UpdateUser(User *User) (err error) {
	err = database.DB.Omit("password").Updates(User).Error
	if err != nil {
		return err
	}
	return nil
}
