// Package account owns user identity: registration, credentials and lookup.
package account

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrNotFound   = errors.New("user not found")
	ErrEmailTaken = errors.New("email already registered")

	// ErrInvalidCredentials covers both "no such email" and "wrong password".
	// Telling the two apart would let anyone check which addresses are
	// registered.
	ErrInvalidCredentials = errors.New("incorrect email or password")
)

const minPasswordLength = 8

type User struct {
	ID           string
	Email        string
	PasswordHash string
	CreatedAt    time.Time
}

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) SignUp(ctx context.Context, email, password string) (User, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return User{}, err
	}
	if len(password) < minPasswordLength {
		return User{}, errors.New("password must be at least 8 characters")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return User{}, err
	}

	return s.repo.Create(ctx, email, string(hash))
}

func (s *Service) LogIn(ctx context.Context, email, password string) (User, error) {
	normalized, err := normalizeEmail(email)
	if err != nil {
		return User{}, ErrInvalidCredentials
	}

	user, err := s.repo.ByEmail(ctx, normalized)
	if errors.Is(err, ErrNotFound) {
		// Hash anyway. Returning early here would make a request for an
		// unknown address measurably faster than one for a known address,
		// which is enough to enumerate registered users with a stopwatch.
		_ = bcrypt.CompareHashAndPassword(dummyHash, []byte(password))
		return User{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return User{}, ErrInvalidCredentials
	}
	return user, nil
}

func (s *Service) ByID(ctx context.Context, id string) (User, error) {
	return s.repo.ByID(ctx, id)
}

// dummyHash is a real bcrypt hash of an unguessable value, used only to burn
// the same amount of time as a genuine comparison.
var dummyHash = []byte("$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy")

func normalizeEmail(raw string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(raw))
	if trimmed == "" {
		return "", errors.New("email is required")
	}
	if _, err := mail.ParseAddress(trimmed); err != nil {
		return "", errors.New("that does not look like an email address")
	}
	return trimmed, nil
}
