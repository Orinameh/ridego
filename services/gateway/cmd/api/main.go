package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/ridego/pkg/proxy"
	"github.com/ridego/services/gateway/internal/config"
	"github.com/ridego/services/gateway/internal/handler"
	"github.com/ridego/services/gateway/internal/middleware"
	"github.com/ridego/services/gateway/internal/ratelimit"
)

// isLocalDev controls whether KubeResolver returns localhost addresses
// (for `make run service=gateway`) or in-cluster Kubernetes DNS names.
func isLocalDev() bool {
	return os.Getenv("RIDEGO_LOCAL_DEV") == "true"
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "err", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(),
		syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	// --- redis — used for rate limiting only --------------------------
	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// --- wire layers ----------------------------------------------------
	// No Consul — KubeResolver uses Kubernetes DNS (or localhost in dev)
	resolver := proxy.NewKubeResolver("ridego", isLocalDev())
	limiter := ratelimit.New(rdb, 100, time.Minute) // 100 req/min per user
	jwtMw := middleware.NewJWT(cfg.JWTPublicKey)

	h := handler.New(resolver, jwtMw, limiter)

	// --- http server ------------------------------------------------------
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("gateway started", "port", cfg.Port, "local_dev", isLocalDev())

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()

	slog.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}
