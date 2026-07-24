package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"
)

const TokenCookieName = "__Secure-token"

type contextKey string

const ClaimsKey contextKey = "claims"

func GetClaims(r *http.Request) *UserClaims {
	claims, _ := r.Context().Value(ClaimsKey).(*UserClaims)
	return claims
}

func WithClaims(r *http.Request, claims *UserClaims) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ClaimsKey, claims))
}

type RegisterUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type MeResponse struct {
	Username string `json:"username"`
}

func RegisterUserHandler(logger *slog.Logger, store AuthStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input RegisterUserInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if input.Username == "" || input.Password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		hashedPassword, err := HashPassword(input.Password)

		if err != nil {
			logger.Error("failed to hash password", "error", err)
			http.Error(w, "Error creating new user", http.StatusInternalServerError)
			return
		}

		user := User{
			Username:     input.Username,
			PasswordHash: hashedPassword,
		}

		if err := store.RegisterUser(r.Context(), &user); err != nil {
			if errors.Is(err, ErrUsernameTaken) {
				http.Error(w, "Username already taken", http.StatusConflict)
				return
			}
			logger.Error("failed to register user", "error", err)
			http.Error(w, "Error creating new user", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}
}

func LoginUserHandler(logger *slog.Logger, store AuthStore, secret string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		r.Body = http.MaxBytesReader(w, r.Body, 4096)
		var input LoginInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if input.Username == "" || input.Password == "" {
			http.Error(w, "Username and password are required", http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUserName(r.Context(), input.Username)
		if err != nil {
			if errors.Is(err, ErrUserNotFound) {
				http.Error(w, "Invalid login credentials", http.StatusUnauthorized)
				return
			}
			logger.Error("failed to look up user", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		if err := VerifyPassword(user.PasswordHash, input.Password); err != nil {
			http.Error(w, "Invalid login credentials", http.StatusUnauthorized)
			return
		}

		token, err := GenerateToken(TokenConfig{
			Username: user.Username,
			Secret:   secret,
			Duration: 24 * time.Hour,
		})
		if err != nil {
			logger.Error("failed to generate token", "error", err)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		http.SetCookie(w, &http.Cookie{
			Name:     TokenCookieName,
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)
	}
}

func Me() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaims(r)

		if claims == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		w.Header().Set("Content-Type", "application/json")

		resp := MeResponse{Username: claims.Subject}
		json.NewEncoder(w).Encode(resp)
	}
}

func LogOutHandler(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     TokenCookieName,
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteStrictMode,
		Secure:   true,
	})
	w.WriteHeader(http.StatusOK)
}
