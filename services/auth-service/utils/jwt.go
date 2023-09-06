package utils

import (
	"auth-service/models"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/gin-gonic/gin"
)

var secret = []byte(os.Getenv("JWT_SECRET"))

func GenerateJWT(user models.User) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"id":   user.ID,
		"role": user.Role,
		"iat":  time.Now().Unix(),
		"eat":  time.Now().Add(24 * time.Hour).Unix(),
	})
	return token.SignedString(secret)
}

func ValidateJWT(c *gin.Context) (jwt.MapClaims, error) {
	token, err := getToken(c)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if ok && token.Valid {
		return claims, nil
	}
	return nil, errors.New("invalid token provided")
}

func CurrentUser(c *gin.Context) models.User {
	claims, err := ValidateJWT(c)
	if err != nil {
		return models.User{}
	}
	userId := uint(claims["id"].(float64))

	user, err := models.GetUserById(userId)
	if err != nil {
		return models.User{}
	}
	return user
}

func getToken(c *gin.Context) (*jwt.Token, error) {
	tokenString := getTokenFromRequest(c)
	token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}

		return secret, nil
	})
	return token, err
}

func getTokenFromRequest(c *gin.Context) string {
	bearerToken := c.Request.Header.Get("Authorization")
	splitToken := strings.Split(bearerToken, " ")
	if len(splitToken) == 2 {
		return splitToken[1]
	}
	return ""
}
