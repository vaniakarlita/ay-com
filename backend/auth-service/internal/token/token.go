package token

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const issuer = "ay-auth"

var ErrInvalid = errors.New("invalid token")

type Manager struct {
	accessSecret  []byte
	refreshSecret []byte
	accessTTL     time.Duration
	refreshTTL    time.Duration
}

func NewManager(accessSecret, refreshSecret string, accessTTL, refreshTTL time.Duration) *Manager {
	return &Manager{
		accessSecret:  []byte(accessSecret),
		refreshSecret: []byte(refreshSecret),
		accessTTL:     accessTTL,
		refreshTTL:    refreshTTL,
	}
}

func (m *Manager) RefreshTTL() time.Duration {
	return m.refreshTTL
}

func (m *Manager) NewAccess(userID string) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.accessTTL)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.accessSecret)
}

func (m *Manager) NewRefresh(userID string) (string, string, error) {
	jti, err := randomID()
	if err != nil {
		return "", "", err
	}
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Issuer:    issuer,
		Subject:   userID,
		ID:        jti,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(m.refreshTTL)),
	}
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.refreshSecret)
	if err != nil {
		return "", "", err
	}
	return signed, jti, nil
}

func (m *Manager) ParseRefresh(raw string) (string, string, error) {
	claims := &jwt.RegisteredClaims{}
	tok, err := jwt.ParseWithClaims(
		raw,
		claims,
		func(*jwt.Token) (any, error) { return m.refreshSecret, nil },
		jwt.WithValidMethods([]string{"HS256"}),
		jwt.WithIssuer(issuer),
	)
	if err != nil || !tok.Valid || claims.Subject == "" || claims.ID == "" {
		return "", "", ErrInvalid
	}
	return claims.Subject, claims.ID, nil
}

func randomID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}