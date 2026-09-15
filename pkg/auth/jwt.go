package auth

import (
	"errors"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/iyertrisha/fleetflow/pkg/httpx"
)

func Secret() []byte {
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "fleetflow-dev-secret-change-me"
	}
	return []byte(s)
}

func Issue(subject string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"sub": subject,
		"iat": time.Now().Unix(),
		"exp": time.Now().Add(ttl).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(Secret())
}

func Parse(token string) (string, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (any, error) {
		if t.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected signing method")
		}
		return Secret(), nil
	})
	if err != nil || !parsed.Valid {
		return "", errors.New("invalid token")
	}
	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}

// Middleware enforces Bearer JWT when AUTH_ENABLED=true.
func Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if os.Getenv("AUTH_ENABLED") != "true" {
			next.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == "/health" {
			next.ServeHTTP(w, r)
			return
		}
		h := r.Header.Get("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			httpx.Error(w, http.StatusUnauthorized, "missing bearer token")
			return
		}
		if _, err := Parse(strings.TrimPrefix(h, "Bearer ")); err != nil {
			httpx.Error(w, http.StatusUnauthorized, "invalid token")
			return
		}
		next.ServeHTTP(w, r)
	})
}
