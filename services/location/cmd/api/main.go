package main

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	pb "github.com/ridego/proto/location"
	"github.com/ridego/services/location/internal/config"
	grpcsrv "github.com/ridego/services/location/internal/grpc"
	"github.com/ridego/services/location/internal/handler"
	"github.com/ridego/services/location/internal/hub"
	"github.com/ridego/services/location/internal/service"
	"github.com/ridego/services/location/internal/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

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

	// --- Redis -----------------------------------------------------
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		Password:     cfg.RedisPassword,
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		slog.Error("redis ping", "err", err)
		os.Exit(1)
	}
	defer rdb.Close()

	// --- wire layers -------------------------------------------------
	geoStore := store.NewRedisGeo(rdb)
	wsHub := hub.New()
	locSvc := service.New(geoStore, wsHub)
	h := handler.New(locSvc, wsHub)

	// --- WebSocket hub — runs its own goroutine ----------------------
	go wsHub.Run(ctx)

	// --- HTTP + WebSocket server --------------------------------------
	httpSrv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      h.Routes(),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // 0 = no timeout, required for long-lived WebSocket conns
	}
	go func() {
		slog.Info("http+ws listening", "port", cfg.Port)
		if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("http", "err", err)
			os.Exit(1)
		}
	}()

	// --- gRPC server — serves NearbyDrivers/GetDriverLocation/UpdateLocation ---
	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		slog.Error("grpc listen", "err", err)
		os.Exit(1)
	}

	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(grpcsrv.LoggingInterceptor),
	)
	pb.RegisterLocationServiceServer(grpcServer, grpcsrv.New(locSvc))
	reflection.Register(grpcServer) // enables grpcurl in dev

	go func() {
		slog.Info("grpc listening", "port", cfg.GRPCPort)
		if err := grpcServer.Serve(lis); err != nil {
			slog.Error("grpc", "err", err)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	grpcServer.GracefulStop()

	shutCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	httpSrv.Shutdown(shutCtx)
}
