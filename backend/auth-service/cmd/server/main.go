package main

import (
	"context"
	"log/slog"
	"net"
	"os"

	"github.com/redis/go-redis/v9"
	"google.golang.org/grpc"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/codestore"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/config"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/db"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/logger"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/mailer"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/service"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/token"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/tokenstore"
)

func main() {
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

	tokens := token.NewManager(cfg.JWTAccessSecret, cfg.JWTRefreshSecret, cfg.AccessTTL, cfg.RefreshTTL)

	mail, err := mailer.NewRabbitMailer(cfg.RabbitURL)
	if err != nil {
		log.Error("rabbitmq connection failed", "error", err)
		os.Exit(1)
	}
	defer mail.Close()
	log.Info("connected to rabbitmq")

	authService := service.New(
		repository.NewUserRepository(pool),
		codestore.New(rdb),
		mail,
		tokenstore.New(rdb),
		tokens,
		log,
	)

	server := grpc.NewServer()
	authpb.RegisterAuthServiceServer(server, authService)
	log.Info("auth-service listening", "port", cfg.GRPCPort, "log_level", cfg.LogLevel)

	if err := server.Serve(lis); err != nil {
		log.Error("grpc server stopped", "error", err)
		os.Exit(1)
	}
}