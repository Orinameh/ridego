package handler

import (
	"net/http"

	pkgMiddleware "github.com/ridego/pkg/middleware"
	"github.com/ridego/pkg/proxy"
	"github.com/ridego/services/gateway/internal/middleware"
	"github.com/ridego/services/gateway/internal/ratelimit"
)

type Handler struct {
	resolver *proxy.KubeResolver
	jwt      *middleware.JWTMiddleware
	limiter  *ratelimit.Limiter
}

func New(r *proxy.KubeResolver, j *middleware.JWTMiddleware, l *ratelimit.Limiter) *Handler {
	return &Handler{resolver: r, jwt: j, limiter: l}
}

// Routes builds the gateway routing table with Go 1.22+ ServeMux.
// Public routes skip JWT; all others go through the full middleware chain.
func (h *Handler) Routes() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	// --- public auth routes (no JWT required) ----------------------
	mux.Handle("POST /v1/auth/register", h.rateLimit(h.proxyTo("user-service")))
	mux.Handle("POST /v1/auth/login", h.rateLimit(h.proxyTo("user-service")))
	mux.Handle("POST /v1/auth/refresh", h.rateLimit(h.proxyTo("user-service")))

	// --- authenticated routes (JWT + rate limit) -------------------
	auth := h.authenticated

	mux.Handle("GET /v1/users/{id}", auth(h.proxyTo("user-service")))
	mux.Handle("PUT /v1/users/{id}", auth(h.proxyTo("user-service")))

	mux.Handle("POST /v1/trips", auth(h.proxyTo("trip-service")))
	mux.Handle("GET /v1/trips/history", auth(h.proxyTo("trip-service")))
	mux.Handle("GET /v1/trips/{id}", auth(h.proxyTo("trip-service")))
	mux.Handle("PUT /v1/trips/{id}/status", auth(h.proxyTo("trip-service")))
	mux.Handle("POST /v1/trips/{id}/rate", auth(h.proxyTo("trip-service")))

	mux.Handle("POST /v1/locations/update", auth(h.proxyTo("location-service")))
	mux.Handle("GET /v1/locations/nearby", auth(h.proxyTo("location-service")))
	mux.Handle("GET /v1/locations/driver/{id}", auth(h.proxyTo("location-service")))
	mux.Handle("POST /v1/locations/driver/{id}/online", auth(h.proxyTo("location-service")))
	mux.Handle("POST /v1/locations/driver/{id}/offline", auth(h.proxyTo("location-service")))

	mux.Handle("POST /v1/match/accept", auth(h.proxyTo("matching-service")))
	mux.Handle("POST /v1/match/reject", auth(h.proxyTo("matching-service")))

	mux.Handle("POST /v1/payments/methods", auth(h.proxyTo("payment-service")))
	mux.Handle("GET /v1/payments/history", auth(h.proxyTo("payment-service")))
	mux.Handle("POST /v1/payments/refund", auth(h.proxyTo("payment-service")))

	// WebSocket — pass-through (JWT validated before upgrade)
	mux.Handle("GET /ws/driver/{id}", auth(h.proxyTo("location-service")))

	return pkgMiddleware.LogMiddleware(pkgMiddleware.RecoveryMiddleware(mux))
}

// authenticated chains: rate limiter → JWT validator → downstream proxy
func (h *Handler) authenticated(next http.Handler) http.Handler {
	return h.rateLimit(h.jwt.Middleware(next))
}

func (h *Handler) rateLimit(next http.Handler) http.Handler {
	return h.limiter.Middleware(next)
}

func (h *Handler) proxyTo(serviceName string) http.Handler {
	return proxy.NewReverseProxy(h.resolver, serviceName)
}
