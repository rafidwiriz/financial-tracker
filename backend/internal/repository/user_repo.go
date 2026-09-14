package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// UserRepo persists users. Every query is user-scoped (D-003); the repo is the
// only place user rows are touched.
type UserRepo struct {
	pool *pgxpool.Pool
}

func NewUserRepo(pool *pgxpool.Pool) *UserRepo {
	return &UserRepo{pool: pool}
}

// EnsureDefaultUser seeds the single-user bootstrap account (D-003, D-018).
// Idempotent — safe to call on every boot.
func (r *UserRepo) EnsureDefaultUser(ctx context.Context, id string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO users (id, email, password_hash, display_name, default_currency)
		VALUES ($1, 'default@local', '', 'Default User', 'USD')
		ON CONFLICT (id) DO NOTHING
	`, id)
	if err != nil {
		return fmt.Errorf("ensure default user: %w", err)
	}
	return nil
}