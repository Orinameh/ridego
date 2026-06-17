package handler

import (
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/ridego/pkg/middleware"
	"github.com/ridego/pkg/response"
	"github.com/ridego/services/payment/internal/service"
)

type Handler struct{ svc *service.PaymentService }

func New(svc *service.PaymentService) *Handler { return &Handler{svc: svc} }

func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// All routes require the X-User-ID header injected by the Gateway
	mux.Handle("POST /v1/payments/methods", middleware.RequireAuthHandler(http.HandlerFunc(h.AddMethod)))
	mux.Handle("GET /v1/payments/history", middleware.RequireAuthHandler(http.HandlerFunc(h.History)))
	mux.Handle("POST /v1/payments/refund", middleware.RequireAuthHandler(http.HandlerFunc(h.Refund)))

	// Apply global middleware (Logging wraps Recovery, so Recovery runs first)
	return middleware.LogMiddleware(middleware.RecoveryMiddleware(mux))
}

func (h *Handler) AddMethod(w http.ResponseWriter, r *http.Request) {
	userID, _ := uuid.Parse(r.Header.Get("X-User-ID"))

	var body struct {
		StripeMethodID string `json:"stripe_method_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.StripeMethodID == "" {
		response.Error(w, http.StatusBadRequest, "stripe_method_id required")
		return
	}

	if err := h.svc.AddPaymentMethod(r.Context(), userID, body.StripeMethodID); err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusCreated, map[string]bool{"ok": true})
}

func (h *Handler) History(w http.ResponseWriter, r *http.Request) {
	userID, _ := uuid.Parse(r.Header.Get("X-User-ID"))

	entries, err := h.svc.GetHistory(r.Context(), userID)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, err.Error())
		return
	}
	response.JSON(w, http.StatusOK, map[string]any{"entries": entries})
}

func (h *Handler) Refund(w http.ResponseWriter, r *http.Request) {
	// TODO: Stripe refund flow with idempotency key
	response.JSON(w, http.StatusAccepted, map[string]string{"status": "queued"})
}
