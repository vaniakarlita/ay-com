package repository

import (
	"context"
	"errors"

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