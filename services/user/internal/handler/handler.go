package handler

import (
	"net/http"

	"github.com/ridego/pkg/middleware"
	"github.com/ridego/services/user/internal/service"
)

type Handler struct{ svc service.UserService }

func New(svc service.UserService) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	// Health + readiness
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Public auth routes — no JWT required
	mux.HandleFunc("POST /v1/auth/register", h.Register)
	mux.HandleFunc("POST /v1/auth/login", h.Login)
	mux.HandleFunc("POST /v1/auth/refresh", h.Refresh)

	// Authenticated routes with path parameter
	mux.Handle("GET /v1/users/{id}", middleware.RequireAuthHandler(http.HandlerFunc(h.GetUser)))
	mux.Handle("PUT /v1/users/{id}", middleware.RequireAuthHandler(http.HandlerFunc(h.UpdateUser)))

	// Apply global middleware (Logging wraps Recovery, so Recovery runs first)
	return middleware.LogMiddleware(middleware.RecoveryMiddleware(mux))
}
