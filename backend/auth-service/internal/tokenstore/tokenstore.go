package tokenstore

import (
	"context"
	"crypto/subtle"
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

func resetKey(email string) string {
	return "reset:" + email
}

func attemptKey(email string) string {
	return "recovery-attempts:" + email
}

func (s *RedisStore) SaveReset(ctx context.Context, email, token string, ttl time.Duration) error {
	return s.rdb.Set(ctx, resetKey(email), token, ttl).Err()
}

func (s *RedisStore) CheckReset(ctx context.Context, email, token string) (bool, error) {
	v, err := s.rdb.Get(ctx, resetKey(email)).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return subtle.ConstantTimeCompare([]byte(v), []byte(token)) == 1, nil
}

func (s *RedisStore) DeleteReset(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, resetKey(email)).Err()
}

func (s *RedisStore) AddAttempt(ctx context.Context, email string, window time.Duration) (int64, error) {
	n, err := s.rdb.Incr(ctx, attemptKey(email)).Result()
	if err != nil {
		return 0, err
	}
	if n == 1 {
		s.rdb.Expire(ctx, attemptKey(email), window)
	}
	return n, nil
}

func (s *RedisStore) Attempts(ctx context.Context, email string) (int64, error) {
	n, err := s.rdb.Get(ctx, attemptKey(email)).Int64()
	if errors.Is(err, redis.Nil) {
		return 0, nil
	}
	return n, err
}

func (s *RedisStore) ClearAttempts(ctx context.Context, email string) error {
	return s.rdb.Del(ctx, attemptKey(email)).Err()
}