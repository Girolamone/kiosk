package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// ErrInvalidToken means the session token was missing, malformed, expired or
// signed with the wrong key. Callers treat all of those the same way: the
// request is anonymous.
var ErrInvalidToken = errors.New("invalid session token")

// TokenIssuer signs and verifies session tokens.
type TokenIssuer struct {
	secret []byte
	ttl    time.Duration
}

func NewTokenIssuer(secret string, ttl time.Duration) *TokenIssuer {
	return &TokenIssuer{secret: []byte(secret), ttl: ttl}
}

func (t *TokenIssuer) TTL() time.Duration { return t.ttl }

func (t *TokenIssuer) Issue(u User) (string, error) {
	now := time.Now()
	claims := jwt.RegisteredClaims{
		Subject:   u.ID,
		ID:        u.Email,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(t.secret)
}

func (t *TokenIssuer) Parse(raw string) (User, error) {
	var claims jwt.RegisteredClaims

	// WithValidMethods pins the algorithm. Without it, a token whose header
	// says alg "none" would parse as validly signed, and anyone could mint
	// their own sessions.
	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || claims.Subject == "" {
		return User{}, ErrInvalidToken
	}

	return User{ID: claims.Subject, Email: claims.ID}, nil
}
