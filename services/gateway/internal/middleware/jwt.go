package middleware

import (
	"context"
	"crypto/rsa"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey string

const claimsKey ctxKey = "claims"

type Claims struct {
	jwt.RegisteredClaims
	UserID string `json:"uid"`
	Role   string `json:"role"`
	Email  string `json:"email"`
}

type JWTMiddleware struct{ publicKey *rsa.PublicKey }

func NewJWT(pub *rsa.PublicKey) *JWTMiddleware { return &JWTMiddleware{publicKey: pub} }

// Middleware validates the Bearer token, then forwards X-User-ID and
// X-User-Role headers downstream so services don't need to re-verify.
func (m *JWTMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if raw == "" {
			http.Error(w, `{"error":"missing token"}`, http.StatusUnauthorized)
			return
		}

		claims := &Claims{}
		_, err := jwt.ParseWithClaims(raw, claims, func(*jwt.Token) (interface{}, error) {
			return m.publicKey, nil
		}, jwt.WithExpirationRequired())

		if err != nil {
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		// Stamp headers so downstream services never touch the JWT
		r = r.WithContext(context.WithValue(r.Context(), claimsKey, claims))
		r.Header.Set("X-User-ID", claims.UserID)
		r.Header.Set("X-User-Role", claims.Role)
		r.Header.Set("X-User-Email", claims.Email)
		// Strip the Authorization header before forwarding
		r.Header.Del("Authorization")

		next.ServeHTTP(w, r)
	})
}
