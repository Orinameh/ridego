package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"
	"github.com/ridego/pkg/middleware"
	"github.com/ridego/pkg/response"
	"github.com/ridego/services/user/internal/models"
	"github.com/ridego/services/user/internal/repository"
)

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Users may only fetch their own profile unless admin
	if !middleware.AuthorizeUser(r, idStr) {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	u, err := h.svc.GetUser(r.Context(), id)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	response.Success(w, http.StatusOK, u)
}

func (h *Handler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	// Fix: Use r.PathValue() for the ID
	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		response.Error(w, http.StatusBadRequest, "invalid user ID")
		return
	}

	// Users may only update their own profile unless admin
	if !middleware.AuthorizeUser(r, idStr) {
		response.Error(w, http.StatusForbidden, "forbidden")
		return
	}

	var req models.UpdateUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	u, err := h.svc.UpdateUser(r.Context(), id, req)
	if errors.Is(err, repository.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "user not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	response.Success(w, http.StatusOK, u)
}
