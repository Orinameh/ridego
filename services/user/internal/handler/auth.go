package handler

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/ridego/pkg/response"
	"github.com/ridego/services/user/internal/models"
	"github.com/ridego/services/user/internal/service"
)

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req models.RegisterRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	if err := req.Validate(); err != nil {
		response.Error(w, http.StatusUnprocessableEntity, err.Error())
		return
	}
	resp, err := h.svc.Register(r.Context(), req)
	if errors.Is(err, service.ErrEmailTaken) {
		response.Error(w, http.StatusConflict, "email already registered")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	response.Success(w, http.StatusCreated, resp)
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req models.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid JSON")
		return
	}

	resp, err := h.svc.Login(r.Context(), req)
	if errors.Is(err, service.ErrInvalidCreds) {
		// Uniform response — don't leak whether email exists
		response.Error(w, http.StatusUnauthorized, "invalid credentials")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	response.Success(w, http.StatusOK, resp)
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.RefreshToken == "" {
		response.Error(w, http.StatusBadRequest, "refresh_token required")
		return
	}

	resp, err := h.svc.RefreshTokens(r.Context(), body.RefreshToken)
	if errors.Is(err, service.ErrInvalidToken) {
		response.Error(w, http.StatusUnauthorized, "invalid or expired refresh token")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "internal error")
		return
	}
	response.Success(w, http.StatusOK, resp)
}
