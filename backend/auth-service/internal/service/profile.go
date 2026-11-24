package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/validator"
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

func (s *AuthService) UpdateProfile(ctx context.Context, req *authpb.UpdateProfileRequest) (*authpb.UpdateProfileResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	params := repository.UpdateProfileParams{
		UserID: req.UserId,
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if err := validator.ValidateName(name); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		params.Name = &name
	}

	if req.Username != nil {
		u := strings.TrimSpace(*req.Username)
		if !validator.UsernameRe.MatchString(u) {
			return nil, status.Error(codes.InvalidArgument, "username must be 3-15 characters: letters, numbers, or underscore")
		}
		params.Username = &u
	}

	if req.Email != nil {
		email := normalizeEmail(*req.Email)
		if !validator.EmailRe.MatchString(email) {
			return nil, status.Error(codes.InvalidArgument, "email must match [name]@[domain].com")
		}
		params.Email = &email
	}

	if req.Gender != nil {
		g := *req.Gender
		if g != "male" && g != "female" {
			return nil, status.Error(codes.InvalidArgument, "gender must be male or female")
		}
		params.Gender = &g
	}

	if req.DateOfBirth != nil {
		dob := strings.TrimSpace(*req.DateOfBirth)
		if err := validator.ValidateAge(dob, time.Now()); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}
		params.DateOfBirth = &dob
	}

	if req.Bio != nil {
		bio := strings.TrimSpace(*req.Bio)
		if len(bio) > 160 {
			return nil, status.Error(codes.InvalidArgument, "bio must be at most 160 characters")
		}
		params.Bio = &bio
	}

	if req.ProfilePictureUrl != nil {
		purl := strings.TrimSpace(*req.ProfilePictureUrl)
		params.ProfilePictureURL = &purl
	}

	if req.BannerUrl != nil {
		burl := strings.TrimSpace(*req.BannerUrl)
		params.BannerURL = &burl
	}

	var passwordChanged bool
	var newAccess, newRefresh string

	if req.NewPassword != nil && strings.TrimSpace(*req.NewPassword) != "" {
		if req.OldPassword == nil || strings.TrimSpace(*req.OldPassword) == "" {
			return nil, status.Error(codes.InvalidArgument, "old password is required to change password")
		}

		attempts, err := s.tokens.PasswordAttempts(ctx, req.UserId)
		if err != nil {
			s.log.Error("read password attempts failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		if attempts >= 5 {
			return nil, status.Error(codes.ResourceExhausted, "too many failed password attempts; try again in 15 minutes")
		}

		cred, err := s.users.FindByID(ctx, req.UserId)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		if err != nil {
			s.log.Error("find user failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}

		if err := bcrypt.CompareHashAndPassword([]byte(cred.PasswordHash), []byte(*req.OldPassword)); err != nil {
			s.tokens.AddPasswordAttempt(ctx, req.UserId, 15*time.Minute)
			return nil, status.Error(codes.InvalidArgument, "incorrect current password")
		}

		if err := validator.ValidatePassword(*req.NewPassword); err != nil {
			return nil, status.Error(codes.InvalidArgument, err.Error())
		}

		newHash, err := bcrypt.GenerateFromPassword([]byte(*req.NewPassword), bcrypt.DefaultCost)
		if err != nil {
			s.log.Error("hash new password failed", "error", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		newHashStr := string(newHash)
		params.PasswordHash = &newHashStr
		passwordChanged = true
	}

	updated, err := s.users.UpdateProfile(ctx, params)
	if err != nil {
		if errors.Is(err, repository.ErrEmailTaken) || errors.Is(err, repository.ErrUsernameTaken) {
			return nil, status.Error(codes.AlreadyExists, err.Error())
		}
		s.log.Error("update profile failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if passwordChanged {
		_ = s.tokens.ClearPasswordAttempts(ctx, req.UserId)
		_ = s.tokens.RevokeAllUserTokens(ctx, req.UserId)

		if s.issuer != nil {
			acc, err := s.issuer.NewAccess(req.UserId)
			if err == nil {
				ref, jti, err := s.issuer.NewRefresh(req.UserId)
				if err == nil {
					_ = s.tokens.Save(ctx, req.UserId, jti, s.issuer.RefreshTTL())
					newAccess = acc
					newRefresh = ref
				}
			}
		}
	}

	resp := &authpb.UpdateProfileResponse{
		Profile: &authpb.GetProfileResponse{
			UserId:            updated.ID,
			Name:              updated.Name,
			Username:          updated.Username,
			ProfilePictureUrl: updated.ProfilePictureURL,
			BannerUrl:         updated.BannerURL,
			Bio:               updated.Bio,
			CreatedAt:         updated.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		},
	}
	if passwordChanged && newAccess != "" && newRefresh != "" {
		resp.NewAccessToken = &newAccess
		resp.NewRefreshToken = &newRefresh
	}

	return resp, nil
}