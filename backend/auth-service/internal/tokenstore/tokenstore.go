package tokenstore

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisStore struct {
	rdb *redis.Client
}

func New(rdb *redis.Client) *RedisStore {
	return &RedisStore{rdb: rdb}
}

func key(userID, jti string) string {
	return "refresh:" + userID + ":" + jti
}

func (s *RedisStore) Save(ctx context.Context, userID, jti string, ttl time.Duration) error {
	return s.rdb.Set(ctx, key(userID, jti), "1", ttl).Err()
}

func (s *RedisStore) Consume(ctx context.Context, userID, jti string) (bool, error) {
	n, err := s.rdb.Del(ctx, key(userID, jti)).Result()
	if err != nil {
		return false, err
	}
	return n == 1, nil
}