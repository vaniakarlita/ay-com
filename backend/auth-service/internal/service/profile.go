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
		BannerUrl:         p.BannerURL,
		Bio:               p.Bio,
		CreatedAt:         p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *AuthService) GetProfileByUsername(ctx context.Context, req *authpb.GetProfileByUsernameRequest) (*authpb.GetProfileResponse, error) {
	p, err := s.users.FindProfileByUsername(ctx, req.Username)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, status.Error(codes.NotFound, "user not found")
	}
	if err != nil {
		s.log.Error("find profile by username failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	return &authpb.GetProfileResponse{
		UserId:            p.ID,
		Name:              p.Name,
		Username:          p.Username,
		ProfilePictureUrl: p.ProfilePictureURL,
		BannerUrl:         p.BannerURL,
		Bio:               p.Bio,
		CreatedAt:         p.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}, nil
}

func (s *AuthService) GetProfilesByIDs(ctx context.Context, req *authpb.GetProfilesByIDsRequest) (*authpb.GetProfilesByIDsResponse, error) {
	list, err := s.users.FindProfilesByIDs(ctx, req.UserIds)
	if err != nil {
		s.log.Error("find profiles by ids failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}
	var res []*authpb.GetProfileResponse
	for _, p := range list {
		res = append(res, &authpb.GetProfileResponse{
			UserId:            p.ID,
			Name:              p.Name,
			Username:          p.Username,
			ProfilePictureUrl: p.ProfilePictureURL,
		})
	}
	return &authpb.GetProfilesByIDsResponse{Profiles: res}, nil
}