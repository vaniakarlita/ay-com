package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/threadpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/config"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/handler"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/logger"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/docs"
	"github.com/joho/godotenv"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/storage"
)

func main() {
	_ = godotenv.Load()
	cfg := config.Load()
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)
	if err := cfg.Validate(); err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	conn, err := grpc.NewClient(cfg.AuthAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("create auth client failed", "error", err)
		os.Exit(1)
	}
	defer conn.Close()

	threadConn, err := grpc.NewClient(cfg.ThreadAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("create thread client failed", "error", err)
		os.Exit(1)
	}
	defer threadConn.Close()

	userConn, err := grpc.NewClient(cfg.UserAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Error("create user client failed", "error", err)
		os.Exit(1)
	}
	defer userConn.Close()

	authClient := authpb.NewAuthServiceClient(conn)
	threadClient := threadpb.NewThreadServiceClient(threadConn)
	userClient := userpb.NewUserServiceClient(userConn)

	mux := http.NewServeMux()
	authHandler := handler.NewAuthHandler(authClient, log)
	authHandler.Routes(mux)
	authHandler.SessionRoutes(mux, middleware.Auth(cfg.JWTAccessSecret))
	authHandler.RecoveryRoutes(mux)
	authHandler.GoogleRoutes(mux)
	docs.Routes(mux)
	handler.NewUploadHandler(storage.New(cfg.SupabaseURL, cfg.SupabaseKey, cfg.SupabaseBucket), log).Routes(mux)

	threadHandler := handler.NewThreadHandler(threadClient, authClient, userClient, log)
	threadHandler.Routes(mux, middleware.Auth(cfg.JWTAccessSecret))

	userHandler := handler.NewUserHandler(userClient, authClient, log)
	userHandler.Routes(mux, middleware.Auth(cfg.JWTAccessSecret))
	userHandler.SettingsRoutes(mux, middleware.Auth(cfg.JWTAccessSecret))

	var root http.Handler = mux
	root = middleware.Logging(log)(root)
	root = middleware.CORS(cfg.AllowedOrigin)(root)
	root = middleware.Recover(log)(root)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           root,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("gateway listening", "port", cfg.Port, "auth_addr", cfg.AuthAddr, "log_level", cfg.LogLevel)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("http server failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Error("shutdown failed", "error", err)
	}
	log.Info("gateway stopped")
}