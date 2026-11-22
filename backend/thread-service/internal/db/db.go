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
    parent_id UUID REFERENCES threads(id) ON DELETE CASCADE,
    scheduled_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE threads ADD COLUMN IF NOT EXISTS parent_id UUID REFERENCES threads(id) ON DELETE CASCADE;

CREATE INDEX IF NOT EXISTS idx_threads_created_at ON threads(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_threads_user_id ON threads(user_id);
CREATE INDEX IF NOT EXISTS idx_threads_is_ad ON threads(is_ad);
CREATE INDEX IF NOT EXISTS idx_threads_parent_id ON threads(parent_id);
CREATE INDEX IF NOT EXISTS idx_threads_author_published ON threads(user_id, created_at DESC, id DESC) WHERE parent_id IS NULL;

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

CREATE TABLE IF NOT EXISTS likes (
    user_id UUID NOT NULL,
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_likes_thread_id ON likes(thread_id);

CREATE TABLE IF NOT EXISTS reposts (
    user_id UUID NOT NULL,
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_reposts_thread_id ON reposts(thread_id);
CREATE INDEX IF NOT EXISTS idx_reposts_created_at ON reposts(created_at DESC);

CREATE TABLE IF NOT EXISTS bookmarks (
    user_id UUID NOT NULL,
    thread_id UUID NOT NULL REFERENCES threads(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, thread_id)
);

CREATE INDEX IF NOT EXISTS idx_bookmarks_user_id ON bookmarks(user_id, created_at DESC);

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
