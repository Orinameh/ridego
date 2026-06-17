package handler

import (
	"net/http"

	"github.com/ridego/pkg/middleware"
	"github.com/ridego/services/trip/internal/service"
)

type Handler struct{ svc service.TripService }

func New(svc service.TripService) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	mux.Handle("GET /v1/trips", middleware.RequireAuthHandler(http.HandlerFunc(h.RequestTrip)))
	mux.Handle("GET /v1/trips/history", middleware.RequireAuthHandler(http.HandlerFunc(h.TripHistory)))
	mux.Handle("GET /v1/trips/{id}", middleware.RequireAuthHandler(http.HandlerFunc(h.GetTrip)))
	mux.Handle("PUT /v1/trips/{id}/status", middleware.RequireAuthHandler(http.HandlerFunc(h.UpdateStatus)))
	mux.Handle("POST /v1/trips/{id}/rate", middleware.RequireAuthHandler(http.HandlerFunc(h.RateTrip)))

	// // Internal endpoint — called by Matching Service, not exposed via gateway.
	// internal := r.PathPrefix("/internal/trips").Subrouter()
	// internal.HandleFunc("/{id}/assign", h.AssignDriver).Methods(http.MethodPost)

	// Apply global middleware (Logging wraps Recovery, so Recovery runs first)
	return middleware.LogMiddleware(middleware.RecoveryMiddleware(mux))
}
