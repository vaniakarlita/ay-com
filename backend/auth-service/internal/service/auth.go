package service

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"errors"
	"fmt"
	"log/slog"
	"math/big"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/validator"
)

const CodeTTL = 5 * time.Minute

type UserStore interface {
	Create(ctx context.Context, u repository.NewUser) (string, error)
	Status(ctx context.Context, email string) (exists bool, active bool, err error)
	Activate(ctx context.Context, email string) error
	FindCredentials(ctx context.Context, email string) (repository.Credentials, error)
	FindByID(ctx context.Context, id string) (repository.Credentials, error)
	FindRecovery(ctx context.Context, email string) (repository.Recovery, error)
	UpdatePassword(ctx context.Context, id, hash string) error
	FindProfile(ctx context.Context, id string) (repository.Profile, error)
}

type CodeStore interface {
	Save(ctx context.Context, email, code string, ttl time.Duration) error
	Get(ctx context.Context, email string) (string, error)
	Delete(ctx context.Context, email string) error
}

type Mailer interface {
	SendVerification(ctx context.Context, to, code string) error
	SendWelcome(ctx context.Context, to string) error
}

type AuthService struct {
	authpb.UnimplementedAuthServiceServer
	users  UserStore
	codes  CodeStore
	mailer Mailer
	tokens TokenStore
	issuer TokenIssuer
	log    *slog.Logger
}

func New(users UserStore, codes CodeStore, mailer Mailer, tokens TokenStore, issuer TokenIssuer, log *slog.Logger) *AuthService {
	return &AuthService{users: users, codes: codes, mailer: mailer, tokens: tokens, issuer: issuer, log: log}
}

func (s *AuthService) Register(ctx context.Context, req *authpb.RegisterRequest) (*authpb.RegisterResponse, error) {
	email := normalizeEmail(req.Email)

	input := validator.Registration{
		Name:             req.Name,
		Username:         req.Username,
		Email:            email,
		Password:         req.Password,
		Gender:           req.Gender,
		DateOfBirth:      req.DateOfBirth,
		ProfileURL:       req.ProfilePictureUrl,
		BannerURL:        req.BannerUrl,
		SecurityQuestion: req.SecurityQuestion,
		SecurityAnswer:   req.SecurityAnswer,
	}
	if err := input.Validate(time.Now()); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hash password failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	answerHash, err := bcrypt.GenerateFromPassword([]byte(normalizeAnswer(req.SecurityAnswer)), bcrypt.DefaultCost)
	if err != nil {
		s.log.Error("hash security answer failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	id, err := s.users.Create(ctx, repository.NewUser{
		Name:               strings.TrimSpace(req.Name),
		Username:           req.Username,
		Email:              email,
		PasswordHash:       string(passwordHash),
		Gender:             req.Gender,
		DateOfBirth:        req.DateOfBirth,
		ProfilePictureURL:  req.ProfilePictureUrl,
		BannerURL:          req.BannerUrl,
		SecurityQuestion:   req.SecurityQuestion,
		SecurityAnswerHash: string(answerHash),
		Newsletter:         req.Newsletter,
	})
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) || errors.Is(err, repository.ErrUsernameTaken) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		s.log.Error("create user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := s.issueCode(ctx, email); err != nil {
		s.log.Error("issue verification code failed", "error", err)
		return nil, status.Error(codes.Internal, "could not send verification code")
	}

	s.log.Info("user registered", "user_id", id)
	return &authpb.RegisterResponse{UserId: id, Message: "verification code sent"}, nil
}

func (s *AuthService) VerifyEmail(ctx context.Context, req *authpb.VerifyEmailRequest) (*authpb.VerifyEmailResponse, error) {
	email := normalizeEmail(req.Email)

	stored, err := s.codes.Get(ctx, email)
	if err != nil {
		s.log.Error("read verification code failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if stored == "" {
		return nil, status.Error(codes.FailedPrecondition, "verification code expired, please request a new one")
	}
	if subtle.ConstantTimeCompare([]byte(stored), []byte(strings.TrimSpace(req.Code))) != 1 {
		return nil, status.Error(codes.InvalidArgument, "invalid verification code")
	}

	if err := s.users.Activate(ctx, email); err != nil {
		s.log.Error("activate user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := s.codes.Delete(ctx, email); err != nil {
		s.log.Warn("delete verification code failed", "error", err)
	}


	if err := s.mailer.SendWelcome(ctx, email); err != nil {
	s.log.Warn("queue welcome email failed", "error", err)
	}

	s.log.Info("account activated")
	return &authpb.VerifyEmailResponse{Message: "account activated"}, nil
}

func (s *AuthService) ResendCode(ctx context.Context, req *authpb.ResendCodeRequest) (*authpb.ResendCodeResponse, error) {
	email := normalizeEmail(req.Email)

	exists, active, err := s.users.Status(ctx, email)
	if err != nil {
		s.log.Error("read user status failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !exists {
		return nil, status.Error(codes.NotFound, "email is not registered")
	}
	if active {
		return nil, status.Error(codes.FailedPrecondition, "account is already active")
	}

	if err := s.issueCode(ctx, email); err != nil {
		s.log.Error("issue verification code failed", "error", err)
		return nil, status.Error(codes.Internal, "could not send verification code")
	}
	return &authpb.ResendCodeResponse{Message: "verification code sent"}, nil
}

func (s *AuthService) issueCode(ctx context.Context, email string) error {
	code, err := generateCode()
	if err != nil {
		return err
	}
	if err := s.codes.Save(ctx, email, code, CodeTTL); err != nil {
		return err
	}
	return s.mailer.SendVerification(ctx, email, code)
}

func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func normalizeAnswer(answer string) string {
	return strings.ToLower(strings.TrimSpace(answer))
}