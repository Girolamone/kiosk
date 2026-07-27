package account

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const uniqueViolation = "23505"

const userColumns = `id::text, email::text, password_hash, created_at`

type Repository struct {
	pool *pgxpool.Pool
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

func (r *Repository) Create(ctx context.Context, email, passwordHash string) (User, error) {
	row := r.pool.QueryRow(ctx, `
		INSERT INTO users (email, password_hash) VALUES ($1, $2)
		RETURNING `+userColumns, email, passwordHash)

	user, err := scanUser(row)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == uniqueViolation {
			return User{}, ErrEmailTaken
		}
		return User{}, err
	}
	return user, nil
}

func (r *Repository) ByEmail(ctx context.Context, email string) (User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE email = $1`, email)
	return scanUser(row)
}

func (r *Repository) ByID(ctx context.Context, id string) (User, error) {
	row := r.pool.QueryRow(ctx, `SELECT `+userColumns+` FROM users WHERE id = $1::uuid`, id)
	return scanUser(row)
}

func scanUser(row interface{ Scan(dest ...any) error }) (User, error) {
	var u User
	err := row.Scan(&u.ID, &u.Email, &u.PasswordHash, &u.CreatedAt)
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		return User{}, ErrNotFound
	case err != nil:
		return User{}, fmt.Errorf("scan user: %w", err)
	}
	return u, nil
}
