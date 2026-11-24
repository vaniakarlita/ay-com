package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrEmailTaken    = errors.New("email is already registered")
	ErrUsernameTaken = errors.New("username is already taken")
	ErrNotFound      = errors.New("user not found")
)

type NewUser struct {
	Name               string
	Username           string
	Email              string
	PasswordHash       string
	Gender             string
	DateOfBirth        string
	ProfilePictureURL  string
	BannerURL          string
	SecurityQuestion   string
	SecurityAnswerHash string
	Newsletter         bool
}

type Credentials struct {
	ID            string
	PasswordHash  string
	IsActive      bool
	IsBanned      bool
	IsDeactivated bool
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, u NewUser) (string, error) {
	var id string
	err := r.pool.QueryRow(ctx, `
		INSERT INTO users (name, username, email, password_hash, gender, date_of_birth,
			profile_picture_url, banner_url, security_question, security_answer_hash, newsletter)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id`,
		u.Name, u.Username, u.Email, u.PasswordHash, u.Gender, u.DateOfBirth,
		u.ProfilePictureURL, u.BannerURL, u.SecurityQuestion, u.SecurityAnswerHash, u.Newsletter,
	).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			switch pgErr.ConstraintName {
			case "users_email_key":
				return "", ErrEmailTaken
			case "users_username_key":
				return "", ErrUsernameTaken
			}
		}
		return "", err
	}
	return id, nil
}

func (r *UserRepository) Status(ctx context.Context, email string) (bool, bool, error) {
	var active bool
	err := r.pool.QueryRow(ctx, `SELECT is_active FROM users WHERE email = $1`, email).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	return true, active, nil
}

func (r *UserRepository) Activate(ctx context.Context, email string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET is_active = TRUE WHERE email = $1`, email)
	return err
}

func (r *UserRepository) FindCredentials(ctx context.Context, email string) (Credentials, error) {
	var c Credentials
	err := r.pool.QueryRow(ctx, `
		SELECT id, password_hash, is_active, is_banned, is_deactivated
		FROM users WHERE email = $1`, email,
	).Scan(&c.ID, &c.PasswordHash, &c.IsActive, &c.IsBanned, &c.IsDeactivated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, err
	}
	return c, nil
}

func (r *UserRepository) FindByID(ctx context.Context, id string) (Credentials, error) {
	var c Credentials
	err := r.pool.QueryRow(ctx, `
		SELECT id, password_hash, is_active, is_banned, is_deactivated
		FROM users WHERE id = $1`, id,
	).Scan(&c.ID, &c.PasswordHash, &c.IsActive, &c.IsBanned, &c.IsDeactivated)
	if errors.Is(err, pgx.ErrNoRows) {
		return Credentials{}, ErrNotFound
	}
	if err != nil {
		return Credentials{}, err
	}
	return c, nil
}

type Recovery struct {
	ID           string
	Question     string
	AnswerHash   string
	PasswordHash string
	IsBanned     bool
}

func (r *UserRepository) FindRecovery(ctx context.Context, email string) (Recovery, error) {
	var v Recovery
	err := r.pool.QueryRow(ctx, `
		SELECT id, security_question, security_answer_hash, password_hash, is_banned
		FROM users WHERE email = $1`, email,
	).Scan(&v.ID, &v.Question, &v.AnswerHash, &v.PasswordHash, &v.IsBanned)
	if errors.Is(err, pgx.ErrNoRows) {
		return Recovery{}, ErrNotFound
	}
	if err != nil {
		return Recovery{}, err
	}
	return v, nil
}

func (r *UserRepository) UpdatePassword(ctx context.Context, id, hash string) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $1 WHERE id = $2`, hash, id)
	return err
}

type UpdateProfileParams struct {
	UserID            string
	Name              *string
	Username          *string
	Email             *string
	Gender            *string
	DateOfBirth       *string
	Bio               *string
	ProfilePictureURL *string
	BannerURL         *string
	PasswordHash      *string
}

func (r *UserRepository) UpdateProfile(ctx context.Context, p UpdateProfileParams) (Profile, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Profile{}, err
	}
	defer tx.Rollback(ctx)

	sets := []string{}
	args := []any{}
	argIdx := 1

	if p.Name != nil {
		sets = append(sets, fmt.Sprintf("name = $%d", argIdx))
		args = append(args, *p.Name)
		argIdx++
	}
	if p.Username != nil {
		sets = append(sets, fmt.Sprintf("username = $%d", argIdx))
		args = append(args, *p.Username)
		argIdx++
	}
	if p.Email != nil {
		sets = append(sets, fmt.Sprintf("email = $%d", argIdx))
		args = append(args, *p.Email)
		argIdx++
	}
	if p.Gender != nil {
		sets = append(sets, fmt.Sprintf("gender = $%d", argIdx))
		args = append(args, *p.Gender)
		argIdx++
	}
	if p.DateOfBirth != nil {
		sets = append(sets, fmt.Sprintf("date_of_birth = $%d", argIdx))
		args = append(args, *p.DateOfBirth)
		argIdx++
	}
	if p.Bio != nil {
		sets = append(sets, fmt.Sprintf("bio = $%d", argIdx))
		args = append(args, *p.Bio)
		argIdx++
	}
	if p.ProfilePictureURL != nil {
		sets = append(sets, fmt.Sprintf("profile_picture_url = $%d", argIdx))
		args = append(args, *p.ProfilePictureURL)
		argIdx++
	}
	if p.BannerURL != nil {
		sets = append(sets, fmt.Sprintf("banner_url = $%d", argIdx))
		args = append(args, *p.BannerURL)
		argIdx++
	}
	if p.PasswordHash != nil {
		sets = append(sets, fmt.Sprintf("password_hash = $%d", argIdx))
		args = append(args, *p.PasswordHash)
		argIdx++
	}

	if len(sets) > 0 {
		query := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(sets, ", "), argIdx)
		args = append(args, p.UserID)
		_, err = tx.Exec(ctx, query, args...)
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				switch pgErr.ConstraintName {
				case "users_email_key":
					return Profile{}, ErrEmailTaken
				case "users_username_key":
					return Profile{}, ErrUsernameTaken
				}
			}
			return Profile{}, err
		}
	}

	var res Profile
	err = tx.QueryRow(ctx, `
		SELECT id, name, username, profile_picture_url, banner_url, bio, created_at
		FROM users WHERE id = $1`, p.UserID,
	).Scan(&res.ID, &res.Name, &res.Username, &res.ProfilePictureURL, &res.BannerURL, &res.Bio, &res.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Profile{}, err
	}

	return res, nil
}

type Profile struct {
	ID                string
	Name              string
	Username          string
	ProfilePictureURL string
	BannerURL         string
	Bio               string
	CreatedAt         time.Time
}

func (r *UserRepository) FindProfile(ctx context.Context, id string) (Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, username, profile_picture_url, banner_url, bio, created_at
		FROM users WHERE id = $1`, id,
	).Scan(&p.ID, &p.Name, &p.Username, &p.ProfilePictureURL, &p.BannerURL, &p.Bio, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (r *UserRepository) FindProfileByUsername(ctx context.Context, username string) (Profile, error) {
	var p Profile
	err := r.pool.QueryRow(ctx, `
		SELECT id, name, username, profile_picture_url, banner_url, bio, created_at
		FROM users 
		WHERE username = $1 AND is_banned = FALSE AND is_deactivated = FALSE`, username,
	).Scan(&p.ID, &p.Name, &p.Username, &p.ProfilePictureURL, &p.BannerURL, &p.Bio, &p.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Profile{}, ErrNotFound
	}
	if err != nil {
		return Profile{}, err
	}
	return p, nil
}

func (r *UserRepository) FindProfilesByIDs(ctx context.Context, ids []string) ([]Profile, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, username, profile_picture_url, banner_url, bio, created_at
		FROM users WHERE id = ANY($1)`, ids,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var profiles []Profile
	for rows.Next() {
		var p Profile
		if err := rows.Scan(&p.ID, &p.Name, &p.Username, &p.ProfilePictureURL, &p.BannerURL, &p.Bio, &p.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, p)
	}
	return profiles, nil
}