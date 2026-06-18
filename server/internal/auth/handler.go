package auth

import (
	"encoding/json"
	"log/slog"
	"net/http"
)

type RegisterUserInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func RegisterUserHandler(logger *slog.Logger, store AuthStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()

		var input RegisterUserInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "Invalid body request", http.StatusBadRequest)
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
