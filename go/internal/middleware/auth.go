package middleware

import (
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"github.com/dict-simulator/go/internal/httputil"
)

// JWTClaims represents the claims in the JWT token
type JWTClaims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Name   string `json:"name"`
	jwt.RegisteredClaims
}

// AuthMiddleware validates JWT tokens and sets X-User-Id header for downstream handlers
func AuthMiddleware(jwtSecret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authorization := r.Header.Get("Authorization")

			if authorization == "" {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header is required")
				return
			}

			// Remove "Bearer " prefix if present
			tokenString := strings.TrimPrefix(authorization, "Bearer ")

			token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (any, error) {
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid or expired token")
				return
			}

			claims, ok := token.Claims.(*JWTClaims)
			if !ok {
				httputil.WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Invalid token claims")
				return
			}

			// Set user ID in request header for downstream handlers
			r.Header.Set("X-User-Id", claims.UserID)

			next.ServeHTTP(w, r)
		})
	}
}
