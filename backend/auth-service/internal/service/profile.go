package service

import (
	"context"
	"errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

func (s *AuthService) GetProfile(ctx context.Context, req *authpb.GetProfileRequest) (*authpb.GetProfileResponse, error) {
	p, err := s.users.FindProfile(ctx, req.UserId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		s.log.Error("find profile failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authpb.GetProfileResponse{
		UserId:            p.ID,
		Name:              p.Name,
		Username:          p.Username,
		ProfilePictureUrl: p.ProfilePictureURL,
	}, nil
}