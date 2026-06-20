package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"time"
)

type RegisterUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterUserHandler(logger *slog.Logger, store AuthStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

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

		var input LoginInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		user, err := store.GetUserByUserName(r.Context(), input.Username)
		if err != nil {
			http.Error(w, "Invalid login credentials", http.StatusUnauthorized)
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
			Name:     "__Secure-token",
			Value:    token,
			HttpOnly: true,
			Path:     "/",
			SameSite: http.SameSiteStrictMode,
			Secure:   true,
		})

		w.WriteHeader(http.StatusOK)
	}

}
