package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/config"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/db"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/logger"
)

func main() {
	cfg := config.Load()
	log := logger.New(cfg.Env, cfg.LogLevel)
	slog.SetDefault(log)

	ctx := context.Background()

	pool, err := db.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Error("database connection failed", "error", err)
		os.Exit(1)
	}
	defer pool.Close()
	log.Info("connected to postgres")

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Error("redis connection failed", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	log.Info("connected to redis")

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}

	server := grpc.NewServer()
	log.Info("auth-service listening", "port", cfg.GRPCPort, "log_level", cfg.LogLevel)

	if err := server.Serve(lis); err != nil {
		log.Error("grpc server stopped", "error", err)
		os.Exit(1)
	}
}