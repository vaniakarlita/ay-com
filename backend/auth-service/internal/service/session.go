package service

import (
	"context"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

type TokenStore interface {
	Save(ctx context.Context, userID, jti string, ttl time.Duration) error
	Consume(ctx context.Context, userID, jti string) (bool, error)
	SaveReset(ctx context.Context, email, token string, ttl time.Duration) error
	CheckReset(ctx context.Context, email, token string) (bool, error)
	DeleteReset(ctx context.Context, email string) error
	AddAttempt(ctx context.Context, email string, window time.Duration) (int64, error)
	Attempts(ctx context.Context, email string) (int64, error)
	ClearAttempts(ctx context.Context, email string) error
}

type TokenIssuer interface {
	NewAccess(userID string) (string, error)
	NewRefresh(userID string) (string, string, error)
	ParseRefresh(raw string) (string, string, error)
	RefreshTTL() time.Duration
}

var dummyHash, _ = bcrypt.GenerateFromPassword([]byte("dummy-password-for-timing"), bcrypt.DefaultCost)

func (s *AuthService) Login(ctx context.Context, req *authpb.LoginRequest) (*authpb.LoginResponse, error) {
	email := normalizeEmail(req.Email)
	if email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	cred, err := s.users.FindCredentials(ctx, email)
	notFound := errors.Is(err, repository.ErrNotFound)
	if err != nil && !notFound {
		s.log.Error("find user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	hash := []byte(cred.PasswordHash)
	if notFound {
		hash = dummyHash
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(req.Password)) != nil || notFound {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	if err := checkAccess(cred); err != nil {
		return nil, err
	}

	access, refresh, err := s.issueTokens(ctx, cred.ID)
	if err != nil {
		return nil, err
	}
	s.log.Info("user logged in", "user_id", cred.ID)
	return &authpb.LoginResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, req *authpb.RefreshTokenRequest) (*authpb.RefreshTokenResponse, error) {
	userID, jti, err := s.issuer.ParseRefresh(req.RefreshToken)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}

	ok, err := s.tokens.Consume(ctx, userID, jti)
	if err != nil {
		s.log.Error("consume refresh token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "refresh token is no longer valid")
	}

	cred, err := s.users.FindByID(ctx, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
	}
	if err != nil {
		s.log.Error("find user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	if err := checkAccess(cred); err != nil {
		return nil, err
	}

	access, refresh, err := s.issueTokens(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &authpb.RefreshTokenResponse{AccessToken: access, RefreshToken: refresh}, nil
}

func (s *AuthService) Logout(ctx context.Context, req *authpb.LogoutRequest) (*authpb.LogoutResponse, error) {
	userID, jti, err := s.issuer.ParseRefresh(req.RefreshToken)
	if err != nil {
		return &authpb.LogoutResponse{Message: "logged out"}, nil
	}
	if _, err := s.tokens.Consume(ctx, userID, jti); err != nil {
		s.log.Error("revoke refresh token failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authpb.LogoutResponse{Message: "logged out"}, nil
}

func (s *AuthService) issueTokens(ctx context.Context, userID string) (string, string, error) {
	access, err := s.issuer.NewAccess(userID)
	if err != nil {
		s.log.Error("sign access token failed", "error", err)
		return "", "", status.Error(codes.Internal, "internal error")
	}
	refresh, jti, err := s.issuer.NewRefresh(userID)
	if err != nil {
		s.log.Error("sign refresh token failed", "error", err)
		return "", "", status.Error(codes.Internal, "internal error")
	}
	if err := s.tokens.Save(ctx, userID, jti, s.issuer.RefreshTTL()); err != nil {
		s.log.Error("store refresh token failed", "error", err)
		return "", "", status.Error(codes.Internal, "internal error")
	}
	return access, refresh, nil
}

func checkAccess(c repository.Credentials) error {
	switch {
	case !c.IsActive:
		return status.Error(codes.PermissionDenied, "account is not verified")
	case c.IsBanned:
		return status.Error(codes.PermissionDenied, "account is banned")
	case c.IsDeactivated:
		return status.Error(codes.PermissionDenied, "account is deactivated")
	}
	return nil
}

