package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type TokenConfig struct {
	Username string
	Secret   string
	Duration time.Duration
}

type UserClaims struct {
	jwt.RegisteredClaims
}

func HashPassword(password string) (string, error) {
	passwordBytes := []byte(password)
	hashedPassword, err := bcrypt.GenerateFromPassword(passwordBytes, bcrypt.DefaultCost)

	if err != nil {
		return "", err
	}

	return string(hashedPassword), nil
}

func VerifyPassword(hashedPass string, inputPass string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPass), []byte(inputPass))
}

func GenerateToken(cfg TokenConfig) (string, error) {
	claims := UserClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   cfg.Username,
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(cfg.Duration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(cfg.Secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

func ValidateToken(tokenString string, secret string) (*UserClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &UserClaims{}, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	return token.Claims.(*UserClaims), nil
}
