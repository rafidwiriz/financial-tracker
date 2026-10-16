package service

import (
	"context"
	"github.com/rafidwiriz/financial-tracker/backend/internal/domain"
	"github.com/rafidwiriz/financial-tracker/backend/internal/repository"
)

type AccountService struct{ repo *repository.AccountRepo }

func NewAccountService(repo *repository.AccountRepo) *AccountService

func (s *AccountService) List(ctx context.Context, userID string) ([]domain.Account, error)
func (s *AccountService) Get(ctx context.Context, userID, id string) (*domain.Account, error)
func (s *AccountService) Create(ctx context.Context, userID string, a *domain.Account) (*domain.Account, error)
func (s *AccountService) Update(ctx context.Context, userID, id string, upd repository.AccountUpdate) (*domain.Account, error)
func (s *AccountService) Close(ctx context.Context, userID, id string) (*domain.Account, error)
