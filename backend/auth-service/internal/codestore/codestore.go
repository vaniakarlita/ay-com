package codestore

import (
	"context"
	"errors"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func key(email string) string {
	return "verify:" + email
}

func (s *RedisStore) Save(ctx context.Context, email, code string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key(email), code, ttl).Err()
}

func (s *RedisStore) Get(ctx context.Context, email string) (string, error) {
	v, err := s.rdb.Get(ctx, key(email)).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}

func (s *RedisStore) Delete(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, key(email)).Err()
}