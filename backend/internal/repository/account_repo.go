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
	Name  *string
	Type  *string
	Color *string
}

func NewAccountRepo(pool *pgxpool.Pool) *AccountRepo {
	return &AccountRepo{pool: pool}
}

func (r *AccountRepo) List(ctx context.Context, userID string) ([]domain.Account, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, type, balance, currency, is_closed, color, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND NOT is_closed
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list accounts: %w", err)
	}
	defer rows.Close()

	accounts := []domain.Account{}
	for rows.Next() {
		var a domain.Account
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt); err != nil {
			return nil, err
		}
		accounts = append(accounts, a)
	}
	return accounts, rows.Err()
}

func (r *AccountRepo) Get(ctx context.Context, userID, id string) (*domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, type, balance, currency, is_closed, color, created_at, updated_at
		FROM accounts
		WHERE user_id = $1 AND id = $2
	`, userID, id).
		Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return &a, nil
}

func (r *AccountRepo) Create(ctx context.Context, account *domain.Account) (*domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx, `
		INSERT INTO accounts (user_id, name, type, currency, color)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, account.UserID, account.Name, account.Type, account.Currency, account.Color).
		Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return &a, nil
}

func (r *AccountRepo) Update(ctx context.Context, account *domain.Account) (*domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx, `
		UPDATE accounts
		SET name = Coalesce($1, name),
			type = Coalesce($2, type),
			color = Coalesce($3, color),
			updated_at = NOW()
		WHERE id = $4 and user_id = $5
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, account.Name, account.Type, account.Color, account.ID, account.UserID).
		Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update account: %w", err)
	}
	return &a, nil
}

func (r *AccountRepo) Close(ctx context.Context, userID, id string) (*domain.Account, error) {
	var a domain.Account
	err := r.pool.QueryRow(ctx, `
		UPDATE accounts
		SET is_closed = true, updated_at = NOW()
		WHERE id = $1 and user_id = $2
		RETURNING id, name, type, balance, currency, is_closed, color, created_at, updated_at
	`, id, userID).
		Scan(&a.ID, &a.Name, &a.Type, &a.Balance, &a.Currency, &a.IsClosed, &a.Color, &a.CreatedAt, &a.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("close account: %w", err)
	}
	return &a, nil
}
