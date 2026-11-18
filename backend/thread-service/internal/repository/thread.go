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
	Categories       []string
}

type CategoryModel struct {
	ID   string
	Name string
}

type ThreadRepository interface {
	Create(ctx context.Context, t *ThreadModel, hashtags []string) error
	GetByID(ctx context.Context, id string) (*ThreadModel, error)
	Delete(ctx context.Context, id, userID string) error
	ListForYou(ctx context.Context, limit int, cursor time.Time) ([]*ThreadModel, error)
	ListCategories(ctx context.Context) ([]CategoryModel, error)
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
			content, reply_permission, community_id, community_name, is_ad, scheduled_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at
	`

	var scheduled *time.Time
	if t.ScheduledAt != nil && !t.ScheduledAt.IsZero() {
		scheduled = t.ScheduledAt
	}

	err = tx.QueryRow(ctx, query,
		t.UserID, t.AuthorName, t.AuthorUsername, t.AuthorAvatarURL, t.AuthorIsVerified,
		t.Content, t.ReplyPermission, t.CommunityID, t.CommunityName, t.IsAd, scheduled,
	).Scan(&t.ID, &t.CreatedAt)
	if err != nil {
		return fmt.Errorf("insert thread: %w", err)
	}

	// Insert categories
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

	// Insert hashtags
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

func (r *pgThreadRepo) GetByID(ctx context.Context, id string) (*ThreadModel, error) {
	query := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.id = $1
		GROUP BY t.id
	`
	t := &ThreadModel{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&t.ID, &t.UserID, &t.AuthorName, &t.AuthorUsername, &t.AuthorAvatarURL, &t.AuthorIsVerified,
		&t.Content, &t.ReplyPermission, &t.CommunityID, &t.CommunityName, &t.IsAd, &t.ScheduledAt, &t.CreatedAt,
		&t.Categories,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("query thread: %w", err)
	}
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

func (r *pgThreadRepo) ListForYou(ctx context.Context, limit int, cursor time.Time) ([]*ThreadModel, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	var rows pgx.Rows
	var err error

	baseQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE (t.scheduled_at IS NULL OR t.scheduled_at <= NOW())
	`

	if cursor.IsZero() {
		query := baseQuery + `
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $1
		`
		rows, err = r.pool.Query(ctx, query, limit)
	} else {
		query := baseQuery + `
			AND t.created_at < $1
			GROUP BY t.id
			ORDER BY t.created_at DESC
			LIMIT $2
		`
		rows, err = r.pool.Query(ctx, query, cursor, limit)
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
			&t.Categories,
		); err != nil {
			return nil, fmt.Errorf("scan thread: %w", err)
		}
		threads = append(threads, t)
	}

	// Fetch an advertisement thread to insert after every 5 threads if available
	adQuery := `
		SELECT 
			t.id, t.user_id, t.author_name, t.author_username, t.author_avatar_url, t.author_is_verified,
			t.content, t.reply_permission, t.community_id, t.community_name, t.is_ad, t.scheduled_at, t.created_at,
			COALESCE(ARRAY_AGG(tc.category_name) FILTER (WHERE tc.category_name IS NOT NULL), '{}') as categories
		FROM threads t
		LEFT JOIN thread_categories tc ON tc.thread_id = t.id
		WHERE t.is_ad = true
		GROUP BY t.id
		ORDER BY t.created_at DESC
		LIMIT 1
	`
	adRow := r.pool.QueryRow(ctx, adQuery)
	ad := &ThreadModel{}
	if err := adRow.Scan(
		&ad.ID, &ad.UserID, &ad.AuthorName, &ad.AuthorUsername, &ad.AuthorAvatarURL, &ad.AuthorIsVerified,
		&ad.Content, &ad.ReplyPermission, &ad.CommunityID, &ad.CommunityName, &ad.IsAd, &ad.ScheduledAt, &ad.CreatedAt,
		&ad.Categories,
	); err == nil && ad.ID != "" {
		// Case requirement: "After every 5 (five) thread, an advertisement thread will be displayed to users."
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
