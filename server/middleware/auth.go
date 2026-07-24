package middleware

import (
	"net/http"

	"github.com/BurrowedInCode/cannabis_coa_analyzer/internal/auth"
)

func AuthMiddleware(secret string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		token, err := r.Cookie(auth.TokenCookieName)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(token.Value, secret)

		if err != nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		next.ServeHTTP(w, auth.WithClaims(r, claims))
	})
}
