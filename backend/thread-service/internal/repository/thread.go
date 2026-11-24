package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ThreadModel struct {
	ID               string
	UserID           string
	AuthorName       string
	AuthorUsername   string
	AuthorAvatarURL  string
	AuthorIsVerified bool
	Content          string
	ReplyPermission  string
	CommunityID      string
	CommunityName    string
	IsAd             bool
	ScheduledAt      *time.Time
	CreatedAt        time.Time
	ParentID         *string
	PinnedAt         *time.Time
	Categories       []string
	LikeCount        int32
	ReplyCount       int32
	RepostCount      int32
	BookmarkCount    int32
	LikedByMe        bool
	RepostedByMe     bool
	BookmarkedByMe   bool
	RepostedByName   string
}

type CategoryModel struct {
	ID   string
	Name string
}

type ThreadRepository interface {
	Create(ctx context.Context, t *ThreadModel, hashtags []string) error
	GetByID(ctx context.Context, id, viewerUserID string) (*ThreadModel, error)
	Delete(ctx context.Context, id, userID string) error
	ListForYou(ctx context.Context, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListFollowing(ctx context.Context, viewerUserID string, authorUserIDs []string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListReplies(ctx context.Context, parentID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListCategories(ctx context.Context) ([]CategoryModel, error)
	Like(ctx context.Context, threadID, userID string) error
	Unlike(ctx context.Context, threadID, userID string) error
	Repost(ctx context.Context, threadID, userID string) error
	Unrepost(ctx context.Context, threadID, userID string) error
	Bookmark(ctx context.Context, threadID, userID string) error
	Unbookmark(ctx context.Context, threadID, userID string) error
	GetCountsAndStatus(ctx context.Context, threadID, userID string) (likes, reposts, bookmarks int32, isLiked, isReposted, isBookmarked bool, err error)
	ListBookmarks(ctx context.Context, userID, query string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListUserThreads(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListUserReplies(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListUserLikes(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error)
	TogglePin(ctx context.Context, threadID, userID string) (bool, error)
	SetPin(ctx context.Context, threadID, userID string, pin bool) (bool, error)
}

type pgThreadRepo struct {
	pool *pgxpool.Pool
}

func NewThreadRepository(pool *pgxpool.Pool) ThreadRepository {
	return &pgThreadRepo{pool: pool}
}

func (r *pgThreadRepo) Create(ctx context.Context, t *ThreadModel, hashtags []string) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO threads (
			user_id, author_name, author_username, author_avatar_url, author_is_verified,
			content, reply_permission, community_id, community_name, is_ad, scheduled_at, parent_id
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, created_at
	`

	var scheduled *time.Time
	if t.ScheduledAt != nil && !t.ScheduledAt.IsZero() {
		scheduled = t.ScheduledAt
	}

	var parentUUID *string
	if t.ParentID != nil && *t.ParentID != "" {
		parentUUID = t.ParentID
	}

	err = tx.QueryRow(ctx, query,
		t.UserID, t.AuthorName, t.AuthorUsername, t.AuthorAvatarURL, t.AuthorIsVerified,
		t.Content, t.ReplyPermission, t.CommunityID, t.CommunityName, t.IsAd, scheduled, parentUUID,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert thread: %w", err)
	}

	for _, cat := range t.Categories {
		cleanCat := strings.TrimSpace(cat)
		if cleanCat == "" {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO thread_categories (thread_id, category_name)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, t.ID, cleanCat)
		if err != nil {
			return fmt.Errorf("insert category: %w", err)
		}
	}

	for _, tag := range hashtags {
		cleanTag := strings.TrimSpace(strings.ToLower(tag))
		if cleanTag == "" {
			continue
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO hashtags (tag) VALUES ($1)
			ON CONFLICT (tag) DO NOTHING
		`, cleanTag)
		if err != nil {
			return fmt.Errorf("upsert hashtag: %w", err)
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO thread_hashtags (thread_id, hashtag)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, t.ID, cleanTag)
		if err != nil {
			return fmt.Errorf("insert thread_hashtag: %w", err)
		}
	}

	return tx.Commit(ctx)
}

func (r *pgThreadRepo) GetByID(ctx context.Context, id, viewerUserID string) (*ThreadModel, error) {
	query := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			t.parent_id,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM threads WHERE parent_id = t.id) as reply_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $2::uuid) END as liked_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $2::uuid) END as reposted_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $2::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.id = $1
		GROUP BY t.id
	`
	t := &ThreadModel{}
	var parentUUID *string
	err := r.pool.QueryRow(ctx, query, id, viewerUserID).Scan(
		&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
		&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
		&parentUUID,
		&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query thread: %w", err)
	}
	t.ParentID = parentUUID
	return t, nil
}

func (r *pgThreadRepo) Delete(ctx context.Context, id, userID string) error {
	res, err := r.pool.Exec(ctx, `DELETE FROM threads WHERE id = $1 AND user_id = $2`, id, userID)
	if err != nil {
		return fmt.Errorf("delete thread: %w", err)
	}
	if res.RowsAffected() == 0 {
		return fmt.Errorf("thread not found or unauthorized")
	}
	return nil
}

func (r *pgThreadRepo) ListForYou(ctx context.Context, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $1::uuid) END as liked_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $1::uuid) END as reposted_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $1::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	if cursor.IsZero() {
		query := baseQuery + `
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, viewerUserID, limit)
	} else {
		query := baseQuery + `
			AND t.created_at < $2
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, viewerUserID, cursor, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query feed: %w", err)
	}
	defer rows.Close()

	var threads []*ThreadModel
	for rows.Next() {
		t := &ThreadModel{}
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&t.Categories, &t.LikeCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		threads = append(threads, t)
	}

	adQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $1::uuid) END as liked_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $1::uuid) END as reposted_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $1::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.is_ad = true
		GROUP BY t.id
		ORDER BY t.created_at DESC
		LIMIT 1
	`
	adRow := r.pool.QueryRow(ctx, adQuery, viewerUserID)
	ad := &ThreadModel{}
	if err := adRow.Scan(
		&ad.ID, &ad.UserID, &ad.AuthorName, &ad.AuthorUsername, &ad.AuthorAvatarURL, &ad.AuthorIsVerified,
		&ad.Content, &ad.ReplyPermission, &ad.CommunityID, &ad.CommunityName, &ad.IsAd, &ad.ScheduledAt, &ad.CreatedAt,
		&ad.Categories, &ad.LikeCount, &ad.RepostCount, &ad.BookmarkCount, &ad.LikedByMe, &ad.RepostedByMe, &ad.BookmarkedByMe,
	); err == nil && ad.ID != "" {
		var combined []*ThreadModel
		for i, th := range threads {
			combined = append(combined, th)
			if (i+1)%5 == 0 && th.ID != ad.ID {
				combined = append(combined, ad)
			}
		}
		threads = combined
	}

	return threads, nil
}

func (r *pgThreadRepo) ListFollowing(ctx context.Context, viewerUserID string, authorUserIDs []string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if len(authorUserIDs) == 0 {
		return nil, nil
	}
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $1::uuid) END as liked_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $1::uuid) END as reposted_by_me,
			CASE WHEN $1 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $1::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.user_id = ANY($2::uuid[])
		  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	if cursor.IsZero() {
		query := baseQuery + `
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, viewerUserID, authorUserIDs, limit)
	} else {
		query := baseQuery + `
			AND t.created_at < $3
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, viewerUserID, authorUserIDs, cursor, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query following feed: %w", err)
	}
	defer rows.Close()

	var threads []*ThreadModel
	for rows.Next() {
		t := &ThreadModel{}
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&t.Categories, &t.LikeCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) ListCategories(ctx context.Context) ([]CategoryModel, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, name FROM categories ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("query categories: %w", err)
	}
	defer rows.Close()

	var list []CategoryModel
	for rows.Next() {
		var c CategoryModel
		if err := rows.Scan(&c.ID, &c.Name); err != nil {
			return nil, err
		}
		list = append(list, c)
	}
	return list, nil
}

func (r *pgThreadRepo) Like(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO likes (user_id, thread_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT DO NOTHING
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) Unlike(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM likes
		WHERE user_id = $1::uuid AND thread_id = $2::uuid
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) Repost(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO reposts (user_id, thread_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT DO NOTHING
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) Unrepost(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM reposts
		WHERE user_id = $1::uuid AND thread_id = $2::uuid
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) Bookmark(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO bookmarks (user_id, thread_id)
		VALUES ($1::uuid, $2::uuid)
		ON CONFLICT DO NOTHING
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) Unbookmark(ctx context.Context, threadID, userID string) error {
	_, err := r.pool.Exec(ctx, `
		DELETE FROM bookmarks
		WHERE user_id = $1::uuid AND thread_id = $2::uuid
	`, userID, threadID)
	return err
}

func (r *pgThreadRepo) GetCountsAndStatus(ctx context.Context, threadID, userID string) (likes, reposts, bookmarks int32, isLiked, isReposted, isBookmarked bool, err error) {
	query := `
		SELECT 
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = $1::uuid),
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = $1::uuid),
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = $1::uuid),
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = $1::uuid AND user_id = $2::uuid) END,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = $1::uuid AND user_id = $2::uuid) END,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = $1::uuid AND user_id = $2::uuid) END
	`
	err = r.pool.QueryRow(ctx, query, threadID, userID).Scan(&likes, &reposts, &bookmarks, &isLiked, &isReposted, &isBookmarked)
	return
}

func (r *pgThreadRepo) ListBookmarks(ctx context.Context, userID, query string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	searchPattern := ""
	if strings.TrimSpace(query) != "" {
		searchPattern = "%" + strings.TrimSpace(query) + "%"
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, b.created_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $1::uuid) as liked_by_me,
			EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $1::uuid) as reposted_by_me,
			true as bookmarked_by_me
		FROM bookmarks b
		JOIN threads t ON t.id = b.thread_id
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE b.user_id = $1::uuid
	`

	if searchPattern != "" {
		baseQuery += ` AND (t.content ILIKE $2 OR t.author_name ILIKE $2 OR t.author_username ILIKE $2)`
		if cursor.IsZero() {
			q := baseQuery + `
				GROUP BY t.id, b.created_at
				ORDER BY b.created_at DESC
				LIMIT $3
			`
			rows, err = r.pool.Query(ctx, q, userID, searchPattern, limit)
		} else {
			q := baseQuery + `
				AND b.created_at < $3
				GROUP BY t.id, b.created_at
				ORDER BY b.created_at DESC
				LIMIT $4
			`
			rows, err = r.pool.Query(ctx, q, userID, searchPattern, cursor, limit)
		}
	} else {
		if cursor.IsZero() {
			q := baseQuery + `
				GROUP BY t.id, b.created_at
				ORDER BY b.created_at DESC
				LIMIT $2
			`
			rows, err = r.pool.Query(ctx, q, userID, limit)
		} else {
			q := baseQuery + `
				AND b.created_at < $2
				GROUP BY t.id, b.created_at
				ORDER BY b.created_at DESC
				LIMIT $3
			`
			rows, err = r.pool.Query(ctx, q, userID, cursor, limit)
		}
	}

	if err != nil {
		return nil, fmt.Errorf("query bookmarks: %w", err)
	}
	defer rows.Close()

	var threads []*ThreadModel
	for rows.Next() {
		t := &ThreadModel{}
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&t.Categories, &t.LikeCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan bookmark: %w", err)
		}
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) ListReplies(ctx context.Context, parentID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			t.parent_id,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM threads WHERE parent_id = t.id) as reply_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $2::uuid) END as liked_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $2::uuid) END as reposted_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $2::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.parent_id = $1::uuid
		  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	if cursor.IsZero() {
		query := baseQuery + `
			GROUP BY t.id
			ORDER BY t.created_at ASC
			LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, parentID, viewerUserID, limit)
	} else {
		query := baseQuery + `
			AND t.created_at > $3
			GROUP BY t.id
			ORDER BY t.created_at ASC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, parentID, viewerUserID, cursor, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query replies: %w", err)
	}
	defer rows.Close()

	var threads []*ThreadModel
	for rows.Next() {
		t := &ThreadModel{}
		var parentUUID *string
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&parentUUID,
			&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan reply: %w", err)
		}
		t.ParentID = parentUUID
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) ListUserThreads(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			t.parent_id, t.pinned_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM threads WHERE parent_id = t.id) as reply_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $2::uuid) END as liked_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $2::uuid) END as reposted_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $2::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.user_id = $1::uuid
		  AND t.parent_id IS NULL
		  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	var threads []*ThreadModel

	if cursor.IsZero() {
		pinnedQuery := baseQuery + `
			AND t.pinned_at IS NOT NULL
			GROUP BY t.id
			ORDER BY t.pinned_at DESC
			LIMIT 1
		`
		pinnedRows, err := r.pool.Query(ctx, pinnedQuery, authorUserID, viewerUserID)
		if err == nil {
			defer pinnedRows.Close()
			if pinnedRows.Next() {
				p := &ThreadModel{}
				var parentUUID *string
				if err := pinnedRows.Scan(
					&p.ID, &p.UserID, &p.AuthorName, &p.AuthorUsername, &p.AuthorAvatarURL, &p.AuthorIsVerified,
					&p.Content, &p.ReplyPermission, &p.CommunityID, &p.CommunityName, &p.IsAd, &p.ScheduledAt, &p.CreatedAt,
					&parentUUID, &p.PinnedAt,
					&p.Categories, &p.LikeCount, &p.ReplyCount, &p.RepostCount, &p.BookmarkCount, &p.LikedByMe, &p.RepostedByMe, &p.BookmarkedByMe,
				); err == nil {
					p.ParentID = parentUUID
					threads = append(threads, p)
				}
			}
		}

		unpinnedQuery := baseQuery + `
			AND t.pinned_at IS NULL
			GROUP BY t.id
			ORDER BY t.created_at DESC, t.id DESC
			LIMIT $3
		`
		rows, err := r.pool.Query(ctx, unpinnedQuery, authorUserID, viewerUserID, limit)
		if err != nil {
			return nil, fmt.Errorf("query user threads: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &ThreadModel{}
			var parentUUID *string
			if err := rows.Scan(
				&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
				&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
				&parentUUID, &t.PinnedAt,
				&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
			); err != nil {
				return nil, fmt.Errorf("scan user thread: %w", err)
			}
			t.ParentID = parentUUID
			threads = append(threads, t)
		}
		return threads, nil
	}

	query := baseQuery + `
		AND t.pinned_at IS NULL
		AND t.created_at < $3
		GROUP BY t.id
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, query, authorUserID, viewerUserID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("query user threads: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		t := &ThreadModel{}
		var parentUUID *string
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&parentUUID, &t.PinnedAt,
			&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan user thread: %w", err)
		}
		t.ParentID = parentUUID
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) ListUserReplies(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			t.parent_id, t.pinned_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM threads WHERE parent_id = t.id) as reply_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $2::uuid) END as liked_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $2::uuid) END as reposted_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $2::uuid) END as bookmarked_by_me
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.user_id = $1::uuid
		  AND t.parent_id IS NOT NULL
		  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	var threads []*ThreadModel

	if cursor.IsZero() {
		pinnedQuery := baseQuery + `
			AND t.pinned_at IS NOT NULL
			GROUP BY t.id
			ORDER BY t.pinned_at DESC
			LIMIT 1
		`
		pinnedRows, err := r.pool.Query(ctx, pinnedQuery, authorUserID, viewerUserID)
		if err == nil {
			defer pinnedRows.Close()
			if pinnedRows.Next() {
				p := &ThreadModel{}
				var parentUUID *string
				if err := pinnedRows.Scan(
					&p.ID, &p.UserID, &p.AuthorName, &p.AuthorUsername, &p.AuthorAvatarURL, &p.AuthorIsVerified,
					&p.Content, &p.ReplyPermission, &p.CommunityID, &p.CommunityName, &p.IsAd, &p.ScheduledAt, &p.CreatedAt,
					&parentUUID, &p.PinnedAt,
					&p.Categories, &p.LikeCount, &p.ReplyCount, &p.RepostCount, &p.BookmarkCount, &p.LikedByMe, &p.RepostedByMe, &p.BookmarkedByMe,
				); err == nil {
					p.ParentID = parentUUID
					threads = append(threads, p)
				}
			}
		}

		unpinnedQuery := baseQuery + `
			AND t.pinned_at IS NULL
			GROUP BY t.id
			ORDER BY t.created_at DESC, t.id DESC
			LIMIT $3
		`
		rows, err := r.pool.Query(ctx, unpinnedQuery, authorUserID, viewerUserID, limit)
		if err != nil {
			return nil, fmt.Errorf("query user replies: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			t := &ThreadModel{}
			var parentUUID *string
			if err := rows.Scan(
				&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
				&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
				&parentUUID, &t.PinnedAt,
				&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
			); err != nil {
				return nil, fmt.Errorf("scan user reply: %w", err)
			}
			t.ParentID = parentUUID
			threads = append(threads, t)
		}
		return threads, nil
	}

	query := baseQuery + `
		AND t.pinned_at IS NULL
		AND t.created_at < $3
		GROUP BY t.id
		ORDER BY t.created_at DESC, t.id DESC
		LIMIT $4
	`
	rows, err := r.pool.Query(ctx, query, authorUserID, viewerUserID, cursor, limit)
	if err != nil {
		return nil, fmt.Errorf("query user replies: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		t := &ThreadModel{}
		var parentUUID *string
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&parentUUID, &t.PinnedAt,
			&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan user reply: %w", err)
		}
		t.ParentID = parentUUID
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) ListUserLikes(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			t.parent_id, t.pinned_at,
			COALESCE(ARRAY_AGG(DISTINCT tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories,
			(SELECT COUNT(*)::int FROM likes WHERE thread_id = t.id) as like_count,
			(SELECT COUNT(*)::int FROM threads WHERE parent_id = t.id) as reply_count,
			(SELECT COUNT(*)::int FROM reposts WHERE thread_id = t.id) as repost_count,
			(SELECT COUNT(*)::int FROM bookmarks WHERE thread_id = t.id) as bookmark_count,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM likes WHERE thread_id = t.id AND user_id = $2::uuid) END as liked_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM reposts WHERE thread_id = t.id AND user_id = $2::uuid) END as reposted_by_me,
			CASE WHEN $2 = '' THEN false ELSE EXISTS (SELECT 1 FROM bookmarks WHERE thread_id = t.id AND user_id = $2::uuid) END as bookmarked_by_me
		FROM likes l
		JOIN threads t ON t.id = l.thread_id
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE l.user_id = $1::uuid
		  AND (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	if cursor.IsZero() {
		query := baseQuery + `
			GROUP BY t.id, l.created_at
			ORDER BY l.created_at DESC
			LIMIT $3
		`
		rows, err = r.pool.Query(ctx, query, targetUserID, viewerUserID, limit)
	} else {
		query := baseQuery + `
			AND l.created_at < $3
			GROUP BY t.id, l.created_at
			ORDER BY l.created_at DESC
			LIMIT $4
		`
		rows, err = r.pool.Query(ctx, query, targetUserID, viewerUserID, cursor, limit)
	}

	if err != nil {
		return nil, fmt.Errorf("query user likes: %w", err)
	}
	defer rows.Close()

	var threads []*ThreadModel
	for rows.Next() {
		t := &ThreadModel{}
		var parentUUID *string
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
			&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
			&parentUUID, &t.PinnedAt,
			&t.Categories, &t.LikeCount, &t.ReplyCount, &t.RepostCount, &t.BookmarkCount, &t.LikedByMe, &t.RepostedByMe, &t.BookmarkedByMe,
		); err != nil {
			return nil, fmt.Errorf("scan liked thread: %w", err)
		}
		t.ParentID = parentUUID
		threads = append(threads, t)
	}

	return threads, nil
}

func (r *pgThreadRepo) TogglePin(ctx context.Context, threadID, userID string) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var parentID *string
	var pinnedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT parent_id, pinned_at 
		FROM threads 
		WHERE id = $1::uuid AND user_id = $2::uuid
	`, threadID, userID).Scan(&parentID, &pinnedAt)
	if err != nil {
		return false, fmt.Errorf("thread not found or unauthorized")
	}

	if pinnedAt != nil {
		_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE id = $1::uuid`, threadID)
		if err != nil {
			return false, fmt.Errorf("unpin failed: %w", err)
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}

	if parentID == nil {
		_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE user_id = $1::uuid AND parent_id IS NULL`, userID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE user_id = $1::uuid AND parent_id IS NOT NULL`, userID)
	}
	if err != nil {
		return false, fmt.Errorf("clear existing pin: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NOW() WHERE id = $1::uuid`, threadID)
	if err != nil {
		return false, fmt.Errorf("pin failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}

func (r *pgThreadRepo) SetPin(ctx context.Context, threadID, userID string, pin bool) (bool, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return false, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var parentID *string
	var pinnedAt *time.Time
	err = tx.QueryRow(ctx, `
		SELECT parent_id, pinned_at 
		FROM threads 
		WHERE id = $1::uuid AND user_id = $2::uuid
	`, threadID, userID).Scan(&parentID, &pinnedAt)
	if err != nil {
		return false, fmt.Errorf("thread not found or unauthorized")
	}

	if !pin {
		if pinnedAt != nil {
			_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE id = $1::uuid`, threadID)
			if err != nil {
				return false, fmt.Errorf("unpin failed: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return false, err
		}
		return false, nil
	}

	if parentID == nil {
		_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE user_id = $1::uuid AND parent_id IS NULL`, userID)
	} else {
		_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NULL WHERE user_id = $1::uuid AND parent_id IS NOT NULL`, userID)
	}
	if err != nil {
		return false, fmt.Errorf("clear existing pin: %w", err)
	}

	_, err = tx.Exec(ctx, `UPDATE threads SET pinned_at = NOW() WHERE id = $1::uuid`, threadID)
	if err != nil {
		return false, fmt.Errorf("pin failed: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return false, err
	}
	return true, nil
}
