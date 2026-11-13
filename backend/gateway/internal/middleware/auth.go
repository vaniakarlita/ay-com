package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type ctxKey struct{}

func UserID(ctx context.Context) string {
	v, _ := ctx.Value(ctxKey{}).(string)
	return v
}

func Auth(secret string) func(http.Handler) http.Handler {
	key := []byte(secret)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			raw, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
			if !found || raw == "" {
				unauthorized(w)
				return
			}

			claims := &jwt.RegisteredClaims{}
			tok, err := jwt.ParseWithClaims(
				raw,
				claims,
				func(*jwt.Token) (any, error) { return key, nil },
				jwt.WithValidMethods([]string{"HS256"}),
				jwt.WithIssuer("ay-auth"),
			)
			if err != nil || !tok.Valid || claims.Subject == "" {
				unauthorized(w)
				return
			}

			ctx := context.WithValue(r.Context(), ctxKey{}, claims.Subject)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, _ = w.Write([]byte(`{"error":"unauthorized"}`))
}