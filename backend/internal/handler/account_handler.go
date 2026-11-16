package handler

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/rafidwiriz/financial-tracker/backend/internal/domain"
	"github.com/rafidwiriz/financial-tracker/backend/internal/dto"
	"github.com/rafidwiriz/financial-tracker/backend/internal/repository"
	"github.com/rafidwiriz/financial-tracker/backend/internal/server"
	"github.com/rafidwiriz/financial-tracker/backend/internal/service"
)

type AccountHandler struct {
	svc  *service.AccountService
	user server.UserResolver
}

func NewAccountHandler(svc *service.AccountService, user server.UserResolver) *AccountHandler {
	return &AccountHandler{svc: svc, user: user}
}

func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actingUser(w, r)
	if !ok {
		return
	}
	accounts, err := h.svc.List(r.Context(), userID)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteData(w, http.StatusOK, accounts)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actingUser(w, r)
	if !ok {
		return
	}
	var req dto.CreateAccountRequest
	if !decode(w, r, &req) {
		return
	}
	if appErr := server.Validate(&req); appErr != nil {
		server.WriteError(w, appErr)
		return
	}
	created, err := h.svc.Create(r.Context(), userID, &domain.Account{
		Name:     req.Name,
		Type:     req.Type,
		Currency: req.Currency,
		Color:    req.Color,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteData(w, http.StatusCreated, created)
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actingUser(w, r)
	if !ok {
		return
	}
	id, ok := accountID(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Get(r.Context(), userID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteData(w, http.StatusOK, a)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actingUser(w, r)
	if !ok {
		return
	}
	id, ok := accountID(w, r)
	if !ok {
		return
	}
	var req dto.UpdateAccountRequest
	if !decode(w, r, &req) {
		return
	}
	if appErr := server.Validate(&req); appErr != nil {
		server.WriteError(w, appErr)
		return
	}
	a, err := h.svc.Update(r.Context(), userID, id, repository.AccountUpdate{
		Name:     req.Name,
		Type:     req.Type,
		Color:    req.Color,
		IsClosed: req.IsClosed,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteData(w, http.StatusOK, a)
}

func (h *AccountHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, ok := h.actingUser(w, r)
	if !ok {
		return
	}
	id, ok := accountID(w, r)
	if !ok {
		return
	}
	a, err := h.svc.Close(r.Context(), userID, id)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	server.WriteData(w, http.StatusOK, a)
}

// --- helpers ---

func (h *AccountHandler) actingUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	userID, err := h.user(r)
	if err != nil {
		server.WriteError(w, server.NewAppError(http.StatusInternalServerError,
			"INTERNAL_ERROR", "could not resolve user", nil))
		return "", false
	}
	return userID, true
}

func accountID(w http.ResponseWriter, r *http.Request) (string, bool) {
	id := chi.URLParam(r, "id")
	if _, err := uuid.Parse(id); err != nil {
		server.WriteError(w, server.NewAppError(http.StatusBadRequest,
			"VALIDATION_ERROR", "invalid account id", nil))
		return "", false
	}
	return id, true
}

func decode[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		server.WriteError(w, server.NewAppError(http.StatusBadRequest,
			"VALIDATION_ERROR", "invalid JSON body", nil))
		return false
	}
	return true
}

func writeServiceError(w http.ResponseWriter, err error) {
	if errors.Is(err, repository.ErrNotFound) {
		server.WriteError(w, server.NewAppError(http.StatusNotFound,
			"NOT_FOUND", "account not found", nil))
		return
	}
	slog.Error("account handler", "error", err)
	server.WriteError(w, server.NewAppError(http.StatusInternalServerError,
		"INTERNAL_ERROR", "internal server error", nil))
}
