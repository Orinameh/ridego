package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	pb "github.com/ridego/proto/location"
	"github.com/ridego/services/matching/internal/config"
	"github.com/ridego/services/matching/internal/handler"
	"github.com/ridego/services/matching/internal/service"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

// locationServiceAddr is the Kubernetes DNS address for the Location
// Service's gRPC port. In local dev, override via LOCATION_GRPC_ADDR env var.
func locationServiceAddr() string {
	if v := os.Getenv("LOCATION_GRPC_ADDR"); v != "" {
		return v
	}
	return "location-service.ridego.svc.cluster.local:50051"
}

// tripServiceAddr is the Kubernetes DNS address for the Trip Service's
// internal HTTP API. In local dev, override via TRIP_SERVICE_ADDR env var.
func tripServiceAddr() string {
	if v := os.Getenv("TRIP_SERVICE_ADDR"); v != "" {
		return v
	}
	return "trip-service.ridego.svc.cluster.local:6002"
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

	// --- gRPC client → Location Service ------------------------------
	grpcConn, err := grpc.NewClient(locationServiceAddr(),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		slog.Error("grpc dial", "err", err)
		os.Exit(1)
	}
	defer grpcConn.Close()
	locClient := pb.NewLocationServiceClient(grpcConn)

	// --- NATS ----------------------------------------------------------
	nc, err := nats.Connect(cfg.NATSUrl,
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1),
	)
	if err != nil {
		slog.Error("nats connect", "err", err)
		os.Exit(1)
	}
	defer nc.Drain()

	// --- trip service HTTP client (internal) ---------------------------
	tripClient := service.NewTripClient(tripServiceAddr())

	// --- wire layers --------------------------------------------------
	svc := service.New(locClient, tripClient, nc)
	h := handler.New(svc)

	// Subscribe to trip.requested events from NATS — runs in its own goroutine
	go svc.ListenForTripRequests(ctx)

	// --- http server -----------------------------------------------------
	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	slog.Info("matching-service started", "port", cfg.Port)

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	srv.Shutdown(shutCtx)
}
