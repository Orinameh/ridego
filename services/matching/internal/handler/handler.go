package handler

import (
	"net/http"

	"github.com/ridego/pkg/middleware"
	"github.com/ridego/pkg/response"
	"github.com/ridego/services/matching/internal/service"
)

type Handler struct{ svc *service.MatchingService }

func New(svc *service.MatchingService) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// Driver responds to a match offer
	mux.HandleFunc("POST /v1/match/accept", h.Accept)
	mux.HandleFunc("POST /v1/match/reject", h.Reject)

	// Apply global middleware (Logging wraps Recovery, so Recovery runs first)
	return middleware.LogMiddleware(middleware.RecoveryMiddleware(mux))
}

func (h *Handler) Accept(w http.ResponseWriter, r *http.Request) {
	driverID := r.Header.Get("X-User-ID")
	if driverID == "" {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.svc.DriverResponse(driverID, true)
	response.JSON(w, http.StatusOK, map[string]bool{"accepted": true})
}

func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	driverID := r.Header.Get("X-User-ID")
	if driverID == "" {
		response.Error(w, http.StatusUnauthorized, "unauthorized")
		return
	}
	h.svc.DriverResponse(driverID, false)
	response.JSON(w, http.StatusOK, map[string]bool{"accepted": false})
}
