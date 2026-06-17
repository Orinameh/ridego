package handler

import (
	"net/http"

	"github.com/ridego/pkg/middleware"
	"github.com/ridego/services/location/internal/hub"
	"github.com/ridego/services/location/internal/service"
)

type Handler struct {
	svc service.LocationService
	hub *hub.Hub
}

func New(svc service.LocationService, h *hub.Hub) *Handler {
	return &Handler{svc: svc, hub: h}
}

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// REST endpoints
	mux.HandleFunc("POST /v1/locations/update", h.UpdateLocation)
	mux.HandleFunc("GET /nearby", h.NearbyDrivers)
	mux.HandleFunc("GET /driver/{id}", h.GetDriverLocation)
	mux.HandleFunc("POST /driver/{id}/online", h.SetOnline)
	mux.HandleFunc("POST /driver/{id}/offline", h.SetOffline)

	// WebSocket — rider app subscribes to a specific driver's position stream
	mux.HandleFunc("GET /ws/driver/{id}", h.DriverStream)

	return middleware.LogMiddleware(middleware.RecoveryMiddleware(mux))
}
