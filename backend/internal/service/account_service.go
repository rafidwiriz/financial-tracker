package service

import (
	"context"

	"github.com/rafidwiriz/financial-tracker/backend/internal/domain"
	"github.com/rafidwiriz/financial-tracker/backend/internal/repository"
)

// AccountService holds account business rules (none beyond delegation yet;
// validation lives in the handler DTOs). It threads the resolved userID —
// D-027 — into every user-scoped repository call.
type AccountService struct {
	repo *repository.AccountRepo
}

func NewAccountService(repo *repository.AccountRepo) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) List(ctx context.Context, userID string) ([]domain.Account, error) {
	return s.repo.List(ctx, userID)
}

func (s *AccountService) Get(ctx context.Context, userID, id string) (*domain.Account, error) {
	return s.repo.Get(ctx, userID, id)
}

func (s *AccountService) Create(ctx context.Context, userID string, a *domain.Account) (*domain.Account, error) {
	return s.repo.Create(ctx, userID, a)
}

func (s *AccountService) Update(ctx context.Context, userID, id string, upd repository.AccountUpdate) (*domain.Account, error) {
	return s.repo.Update(ctx, userID, id, upd)
}

func (s *AccountService) Close(ctx context.Context, userID, id string) (*domain.Account, error) {
	return s.repo.Close(ctx, userID, id)
}
