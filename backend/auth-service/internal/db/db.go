package db

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const schema = `
CREATE TABLE IF NOT EXISTS users (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	name VARCHAR(100) NOT NULL,
	username VARCHAR(15) NOT NULL UNIQUE,
	email VARCHAR(255) NOT NULL UNIQUE,
	password_hash TEXT NOT NULL,
	gender VARCHAR(6) NOT NULL CHECK (gender IN ('male', 'female')),
	date_of_birth DATE NOT NULL,
	profile_picture_url TEXT NOT NULL,
	banner_url TEXT NOT NULL,
	security_question TEXT NOT NULL,
	security_answer_hash TEXT NOT NULL,
	newsletter BOOLEAN NOT NULL DEFAULT FALSE,
	is_active BOOLEAN NOT NULL DEFAULT FALSE,
	is_banned BOOLEAN NOT NULL DEFAULT FALSE,
	is_deactivated BOOLEAN NOT NULL DEFAULT FALSE,
	bio TEXT NOT NULL DEFAULT '',
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE users ADD COLUMN IF NOT EXISTS bio TEXT NOT NULL DEFAULT '';
`

func Connect(ctx context.Context, url string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("create pool: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}

	if _, err := pool.Exec(ctx, schema); err != nil {
		pool.Close()
		return nil, fmt.Errorf("migrate schema: %w", err)
	}

	return pool, nil
}