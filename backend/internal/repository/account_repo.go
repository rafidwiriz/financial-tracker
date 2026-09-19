package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/rafidwiriz/financial-tracker/backend/internal/domain"
)

type AccountRepo struct {
	pool *pgxpool.Pool
}

type AccountUpdate struct {
	Name     *string
	Type     *string
	Color    *string
	IsClosed *bool
}

const accountColumns = "id, name, type, balance, currency, is_closed, color, created_at, updated_at"

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

// rowScanner is satisfied by both pgx.Row and pgx.Rows.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanAccount(row rowScanner) (*domain.Account, error) {
	var a domain.Account
	if err := row.Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency,
		&a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt); err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *AccountRepo) List(ctx context.Context, userID string) ([]domain.Account, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT `+accountColumns+`
		FROM accounts
		WHERE user_id = $1 AND NOT is_closed
		ORDER BY created_at DESC, id DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	accounts := []domain.Account{}
	for rows.Next() {
		a, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("list accounts: %w", err)
		}
		accounts = append(accounts, *a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	return accounts, nil
}

func (r *AccountRepo) Get(ctx context.Context, userID, id string) (*domain.Account, error) {
	a, err := scanAccount(r.pool.QueryRow(ctx, `
		SELECT `+accountColumns+`
		FROM accounts
		WHERE user_id = $1 AND id = $2
	`, userID, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return a, nil
}

func (r *AccountRepo) Create(ctx context.Context, userID string, account *domain.Account) (*domain.Account, error) {
	a, err := scanAccount(r.pool.QueryRow(ctx, `
		INSERT INTO accounts (user_id, name, type, currency, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, userID, account.Name, account.Type, account.Currency, account.Color))
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return a, nil
}

func (r *AccountRepo) Update(ctx context.Context, userID, id string, upd AccountUpdate) (*domain.Account, error) {
	a, err := scanAccount(r.pool.QueryRow(ctx, `
		UPDATE accounts
		SET name = COALESCE($1, name),
			type = COALESCE($2, type),
			color = COALESCE($3, color),
			is_closed = COALESCE($4, is_closed),
			updated_at = NOW()
		WHERE id = $5 AND user_id = $6
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, upd.Name, upd.Type, upd.Color, upd.IsClosed, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	return a, nil
}

func (r *AccountRepo) Close(ctx context.Context, userID, id string) (*domain.Account, error) {
	a, err := scanAccount(r.pool.QueryRow(ctx, `
		UPDATE accounts
		SET is_closed = true, updated_at = NOW()
		WHERE id = $1 AND user_id = $2
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, id, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("close account: %w", err)
	}
	return a, nil
}
