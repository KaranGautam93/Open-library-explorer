package middleware

import (
	"context"
	"net/http"
	"open-library-explorer/internal/utils"
	"strings"
)

type contextKey string

const ContextUserID contextKey = "user_id"

func JWTAuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			utils.JSONError(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		claims, err := utils.ParseJWT(tokenStr)
		if err != nil {
			utils.JSONError(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), ContextUserID, claims.UserID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
