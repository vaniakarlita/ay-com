package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/vaniakarlita/ay-com/backend/user-service/internal/config"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/db"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/logger"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/service"
	"github.com/vaniakarlita/ay-com/backend/user-service/userpb"
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
	log.Info("connected to user postgres")

	var rdb *redis.Client
	if cfg.RedisAddr != "" {
		rdb = redis.NewClient(&redis.Options{Addr: cfg.RedisAddr})
		if err := rdb.Ping(ctx).Err(); err != nil {
			log.Warn("redis ping failed, continuing without cache", "error", err)
		} else {
			log.Info("connected to redis")
		}
	}

	lis, err := net.Listen("tcp", ":"+cfg.GRPCPort)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}

	repo := repository.NewFollowRepository(pool)
	userSvc := service.NewUserService(repo, rdb, log)

	server := grpc.NewServer()
	userpb.RegisterUserServiceServer(server, userSvc)

	log.Info("user-service listening", "port", cfg.GRPCPort, "log_level", cfg.LogLevel)

	if err := server.Serve(lis); err != nil {
		log.Error("grpc server stopped", "error", err)
		os.Exit(1)
	}
}
