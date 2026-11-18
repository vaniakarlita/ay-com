package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS threads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    author_name VARCHAR(255) NOT NULL,
    author_username VARCHAR(100) NOT NULL,
    author_avatar_url TEXT NOT NULL DEFAULT '',
    author_is_verified BOOLEAN NOT NULL DEFAULT false,
    content TEXT NOT NULL,
    reply_permission VARCHAR(50) NOT NULL DEFAULT 'everyone',
    community_id VARCHAR(100) NOT NULL DEFAULT '',
    community_name VARCHAR(255) NOT NULL DEFAULT '',
    is_ad BOOLEAN NOT NULL DEFAULT false,
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_threads_created_at ON threads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_user_id ON threads(user_id);
CREATE INDEX IF NOT EXISTS idx_threads_is_ad ON threads(is_ad);

CREATE TABLE IF NOT EXISTS categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS thread_categories (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    category_name VARCHAR(100) NOT NULL,
    PRIMARY KEY (thread_id, category_name)
);

CREATE TABLE IF NOT EXISTS hashtags (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tag VARCHAR(100) NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS thread_hashtags (
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    hashtag VARCHAR(100) NOT NULL,
    PRIMARY KEY (thread_id, hashtag)
);

-- Seed default categories if empty
INSERT INTO categories (name) VALUES 
    ('Technology'),
    ('Gaming'),
    ('Sports'),
    ('Entertainment'),
    ('News'),
    ('Art & Design')
ON CONFLICT (name) DO NOTHING;
`

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.MaxConns = 15
	cfg.MinConns = 2
	cfg.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, fmt.Errorf("new pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate: %w", err)
	}

	return pool, nil
}
