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

	// Pin the accepted algorithm.
	//
	// This is not what stops an alg "none" token today: jwt/v5 already
	// refuses those unless you hand it the library's explicit unsafe sentinel
	// as the key. The pin earns its place against algorithm confusion, which
	// the library cannot rule out on its own. If this ever moves to RS256,
	// a key function handing back a public key would otherwise let an
	// attacker sign HS256 using that public key as the shared secret — the
	// key is public, so anyone can. Pinning the algorithm here means the
	// server decides, not the token.
	_, err := jwt.ParseWithClaims(raw, &claims,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || claims.Subject == "" {
		return User{}, ErrInvalidToken
	}

	return User{ID: claims.Subject, Email: claims.ID}, nil
}
