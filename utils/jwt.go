package utils

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTClaims struct {
	UserID    string `json:"sub"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	SessionID string `json:"sid"`
	jwt.RegisteredClaims
}

func getJWTSecret() ([]byte, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, errors.New("JWT_SECRET is strictly required in environment variables")
	}
	return []byte(secret), nil
}

func GenerateAccessToken(userID, email, name, sessionID string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	expiryMinutes, _ := strconv.Atoi(os.Getenv("JWT_EXPIRY_MINUTES"))
	if expiryMinutes <= 0 {
		expiryMinutes = 120
	}

	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		Email:     email,
		Name:      name,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func GenerateRefreshToken(userID, sessionID string) (string, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return "", err
	}

	expiryMinutes, _ := strconv.Atoi(os.Getenv("JWT_REFRESH_EXPIRY_MINUTES"))
	if expiryMinutes <= 0 {
		expiryMinutes = 43200 // 30 days
	}

	now := time.Now()
	claims := JWTClaims{
		UserID:    userID,
		SessionID: sessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiryMinutes) * time.Minute)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(secret)
}

func ParseJWTToken(tokenString string) (*JWTClaims, error) {
	secret, err := getJWTSecret()
	if err != nil {
		return nil, err
	}

	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return secret, nil
	})

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}
