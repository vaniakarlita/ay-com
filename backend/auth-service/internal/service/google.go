package service

import (
	"context"
	"errors"
	"os"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

const defaultGoogleClientID = "138897984454-b4ug0970501074e8m9bmroo5bdfnf79v.apps.googleusercontent.com"

func googleClientID() string {
	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		return v
	}
	return defaultGoogleClientID
}

func (s *AuthService) GoogleAuth(ctx context.Context, req *authpb.GoogleAuthRequest) (*authpb.GoogleAuthResponse, error) {
	if req.IdToken == "" {
		return nil, status.Error(codes.InvalidArgument, "google token is required")
	}

	payload, err := idtoken.Validate(ctx, req.IdToken, googleClientID())
	if err != nil {
		s.log.Warn("invalid google token", "error", err)
		return nil, status.Error(codes.Unauthenticated, "invalid google token")
	}

	rawEmail, _ := payload.Claims["email"].(string)
	verified, _ := payload.Claims["email_verified"].(bool)
	name, _ := payload.Claims["name"].(string)
	email := normalizeEmail(rawEmail)
	if email == "" || !verified {
		return nil, status.Error(codes.Unauthenticated, "google email is not verified")
	}

	cred, err := s.users.FindCredentials(ctx, email)
	if errors.Is(err, repository.ErrNotFound) {
		return &authpb.GoogleAuthResponse{Registered: false, Email: email, Name: name}, nil
	}
	if err != nil {
		s.log.Error("find user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := checkAccess(cred); err != nil {
		return nil, err
	}

	access, refresh, err := s.issueTokens(ctx, cred.ID)
	if err != nil {
		return nil, err
	}
	s.log.Info("user logged in with google", "user_id", cred.ID)
	return &authpb.GoogleAuthResponse{
		Registered:   true,
		AccessToken:  access,
		RefreshToken: refresh,
		Email:        email,
		Name:         name,
	}, nil
}