package repository

import (
	"context"
	"errors"
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
	Block(ctx context.Context, blockerID, blockedID string) error
	Unblock(ctx context.Context, blockerID, blockedID string) error
	CheckBlock(ctx context.Context, userA, userB string) (isBlocking, isBlockedBy bool, err error)
	CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error
	ListBlockedUsers(ctx context.Context, blockerID string, limit int, cursor time.Time) ([]RelationItem, error)
	GetBlockedUserIDs(ctx context.Context, userID string) ([]string, error)
	GetNotificationSettings(ctx context.Context, userID string) (like, repost, follow, mention, community bool, err error)
	UpdateNotificationSettings(ctx context.Context, userID string, like, repost, follow, mention, community bool) error
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
		  AND ($2 = '' OR NOT EXISTS (
		      SELECT 1 FROM blocks b 
		      WHERE (b.blocker_id = $2::uuid AND b.blocked_id = f.follower_id)
		         OR (b.blocker_id = f.follower_id AND b.blocked_id = $2::uuid)
		  ))
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
		  AND ($2 = '' OR NOT EXISTS (
		      SELECT 1 FROM blocks b 
		      WHERE (b.blocker_id = $2::uuid AND b.blocked_id = f.followee_id)
		         OR (b.blocker_id = f.followee_id AND b.blocked_id = $2::uuid)
		  ))
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

func (r *pgFollowRepo) Block(ctx context.Context, blockerID, blockedID string) error {
	if blockerID == blockedID {
		return fmt.Errorf("cannot block self")
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO blocks (blocker_id, blocked_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT DO NOTHING
	`, blockerID, blockedID)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		DELETE FROM follows
		WHERE (follower_id = $1::uuid AND followee_id = $2::uuid)
		   OR (follower_id = $2::uuid AND followee_id = $1::uuid)
	`, blockerID, blockedID)
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *pgFollowRepo) Unblock(ctx context.Context, blockerID, blockedID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM blocks
		WHERE blocker_id = $1::uuid AND blocked_id = $2::uuid
	`, blockerID, blockedID)
	return err
}

func (r *pgFollowRepo) CheckBlock(ctx context.Context, userA, userB string) (isBlocking, isBlockedBy bool, err error) {
	if userA == "" || userB == "" || userA == userB {
		return false, false, nil
	}

	query := `
		SELECT
			EXISTS (SELECT 1 FROM blocks WHERE blocker_id = $1::uuid AND blocked_id = $2::uuid),
			EXISTS (SELECT 1 FROM blocks WHERE blocker_id = $2::uuid AND blocked_id = $1::uuid)
	`
	err = r.pool.QueryRow(ctx, query, userA, userB).Scan(&isBlocking, &isBlockedBy)
	return
}

func (r *pgFollowRepo) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if reporterID == reportedID {
		return fmt.Errorf("cannot report self")
	}

	_, err := r.pool.Exec(ctx, `
		INSERT INTO reports (reporter_id, reported_id, reason, details)
		VALUES ($1::uuid, $2::uuid, $3, $4)
	`, reporterID, reportedID, reason, details)
	return err
}

func (r *pgFollowRepo) ListBlockedUsers(ctx context.Context, blockerID string, limit int, cursor time.Time) ([]RelationItem, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			b.blocked_id::text,
			b.created_at,
			false as is_following
		FROM blocks b
		WHERE b.blocker_id = $1::uuid
	`

	if cursor.IsZero() {
		q := baseQuery + ` ORDER BY b.created_at DESC LIMIT $2`
		rows, err = r.pool.Query(ctx, q, blockerID, limit)
	} else {
		q := baseQuery + ` AND b.created_at < $2 ORDER BY b.created_at DESC LIMIT $3`
		rows, err = r.pool.Query(ctx, q, blockerID, cursor, limit)
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

func (r *pgFollowRepo) GetBlockedUserIDs(ctx context.Context, userID string) ([]string, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT blocked_id::text FROM blocks WHERE blocker_id = $1::uuid
		UNION
		SELECT blocker_id::text FROM blocks WHERE blocked_id = $1::uuid
	`, userID)
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

func (r *pgFollowRepo) GetNotificationSettings(ctx context.Context, userID string) (like, repost, follow, mention, community bool, err error) {
	query := `
		SELECT notif_like, notif_repost, notif_follow, notif_mention, notif_community
		FROM notification_settings
		WHERE user_id = $1::uuid
	`
	err = r.pool.QueryRow(ctx, query, userID).Scan(&like, &repost, &follow, &mention, &community)
	if errors.Is(err, pgx.ErrNoRows) {
		return true, true, true, true, true, nil
	}
	return
}

func (r *pgFollowRepo) UpdateNotificationSettings(ctx context.Context, userID string, like, repost, follow, mention, community bool) error {
	query := `
		INSERT INTO notification_settings (user_id, notif_like, notif_repost, notif_follow, notif_mention, notif_community, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			notif_like = EXCLUDED.notif_like,
			notif_repost = EXCLUDED.notif_repost,
			notif_follow = EXCLUDED.notif_follow,
			notif_mention = EXCLUDED.notif_mention,
			notif_community = EXCLUDED.notif_community,
			updated_at = NOW()
	`
	_, err := r.pool.Exec(ctx, query, userID, like, repost, follow, mention, community)
	return err
}
