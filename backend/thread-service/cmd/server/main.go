package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"
	"google.golang.org/grpc"

	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/config"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/db"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/logger"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/service"
	"github.com/vaniakarlita/ay-com/backend/thread-service/threadpb"
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

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("connected to thread postgres")

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}

	repo := repository.NewThreadRepository(pool)
	threadSvc := service.NewThreadService(repo, log)

	server := grpc.NewServer()
	threadpb.RegisterThreadServiceServer(server, threadSvc)

	log.Info("thread-service listening", "port", cfg.GRPCPort, "log_level", cfg.LogLevel)

	if err := server.Serve(lis); err != nil {
		log.Error("grpc server stopped", "error", err)
		os.Exit(1)
	}
}
