package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type RelationItem struct {
	UserID      string
	CreatedAt   time.Time
	IsFollowing bool
}

type SuggestedAccount struct {
	UserID        string
	FollowerCount int32
}

type FollowRepository interface {
	Follow(ctx context.Context, followerID, followeeID string) error
	Unfollow(ctx context.Context, followerID, followeeID string) error
	GetStats(ctx context.Context, targetUserID, viewerUserID string) (followers, following int32, isFollowing bool, err error)
	ListFollowingIDs(ctx context.Context, userID string, limit int) ([]string, error)
	ListFollowers(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]RelationItem, error)
	ListFollowing(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]RelationItem, error)
	SuggestAccounts(ctx context.Context, userID string, limit int) ([]SuggestedAccount, error)
}

type pgFollowRepo struct {
	pool *pgxpool.Pool
}

func NewFollowRepository(pool *pgxpool.Pool) FollowRepository {
	return &pgFollowRepo{pool: pool}
}

func (r *pgFollowRepo) Follow(ctx context.Context, followerID, followeeID string) error {
	if followerID == followeeID {
		return fmt.Errorf("cannot follow self")
	}
	_, err := r.pool.Exec(ctx, `
		INSERT INTO follows (follower_id, followee_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT DO NOTHING
	`, followerID, followeeID)
	return err
}

func (r *pgFollowRepo) Unfollow(ctx context.Context, followerID, followeeID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM follows
		WHERE follower_id = $1::uuid AND followee_id = $2::uuid
	`, followerID, followeeID)
	return err
}

func (r *pgFollowRepo) GetStats(ctx context.Context, targetUserID, viewerUserID string) (followers, following int32, isFollowing bool, err error) {
	query := `
		SELECT 
			(SELECT COUNT(*)::int FROM follows WHERE followee_id = $1::uuid),
			(SELECT COUNT(*)::int FROM follows WHERE follower_id = $1::uuid),
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM follows WHERE follower_id = $2::uuid AND followee_id = $1::uuid) END
	`
	err = r.pool.QueryRow(ctx, query, targetUserID, viewerUserID).Scan(&followers, &following, &isFollowing)
	return
}

func (r *pgFollowRepo) ListFollowingIDs(ctx context.Context, userID string, limit int) ([]string, error) {
	if limit <= 0 || limit > 500 {
		limit = 500
	}
	rows, err := r.pool.Query(ctx, `
		SELECT followee_id::text
		FROM follows
		WHERE follower_id = $1::uuid
		ORDER BY created_at DESC
		LIMIT $2
	`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (r *pgFollowRepo) ListFollowers(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]RelationItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			f.follower_id::text,
			f.created_at,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM follows WHERE follower_id = $2::uuid AND followee_id = f.follower_id) END as is_following
		FROM follows f
		WHERE f.followee_id = $1::uuid
	`

	if cursor.IsZero() {
		q := baseQuery + ` ORDER BY f.created_at DESC LIMIT $3`
		rows, err = r.pool.Query(ctx, q, targetUserID, viewerUserID, limit)
	} else {
		q := baseQuery + ` AND f.created_at < $3 ORDER BY f.created_at DESC LIMIT $4`
		rows, err = r.pool.Query(ctx, q, targetUserID, viewerUserID, cursor, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RelationItem
	for rows.Next() {
		var it RelationItem
		if err := rows.Scan(&it.UserID, &it.CreatedAt, &it.IsFollowing); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *pgFollowRepo) ListFollowing(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]RelationItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			f.followee_id::text,
			f.created_at,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM follows WHERE follower_id = $2::uuid AND followee_id = f.followee_id) END as is_following
		FROM follows f
		WHERE f.follower_id = $1::uuid
	`

	if cursor.IsZero() {
		q := baseQuery + ` ORDER BY f.created_at DESC LIMIT $3`
		rows, err = r.pool.Query(ctx, q, targetUserID, viewerUserID, limit)
	} else {
		q := baseQuery + ` AND f.created_at < $3 ORDER BY f.created_at DESC LIMIT $4`
		rows, err = r.pool.Query(ctx, q, targetUserID, viewerUserID, cursor, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []RelationItem
	for rows.Next() {
		var it RelationItem
		if err := rows.Scan(&it.UserID, &it.CreatedAt, &it.IsFollowing); err != nil {
			return nil, err
		}
		items = append(items, it)
	}
	return items, nil
}

func (r *pgFollowRepo) SuggestAccounts(ctx context.Context, userID string, limit int) ([]SuggestedAccount, error) {
	if limit <= 0 || limit > 10 {
		limit = 3
	}
	query := `
		SELECT 
			f.followee_id::text,
			COUNT(*)::int as follower_count
		FROM follows f
		WHERE ($1 = '' OR (f.followee_id != $1::uuid AND f.followee_id NOT IN (
			SELECT followee_id FROM follows WHERE follower_id = $1::uuid
		)))
		GROUP BY f.followee_id
		ORDER BY follower_count DESC
		LIMIT $2
	`
	rows, err := r.pool.Query(ctx, query, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []SuggestedAccount
	for rows.Next() {
		var a SuggestedAccount
		if err := rows.Scan(&a.UserID, &a.FollowerCount); err != nil {
			return nil, err
		}
		list = append(list, a)
	}
	return list, nil
}
