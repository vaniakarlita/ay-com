package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/vaniakarlita/ay-com/backend/auth-service/authpb"
	"github.com/vaniakarlita/ay-com/backend/auth-service/internal/repository"
)

func TestGetProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("returns the profile", func(t *testing.T) {
		f := newFixture()
		f.users.On("FindProfile", mock.Anything, "u1").Return(
			repository.Profile{ID: "u1", Name: "Adrian Yu", Username: "adrian_yu", ProfilePictureURL: "https://example.com/p.png"}, nil)

		res, err := f.svc.GetProfile(ctx, &authpb.GetProfileRequest{UserId: "u1"})

		require.NoError(t, err)
		assert.Equal(t, "Adrian Yu", res.Name)
		assert.Equal(t, "adrian_yu", res.Username)
		assert.Equal(t, "https://example.com/p.png", res.ProfilePictureUrl)
	})

	t.Run("unknown user", func(t *testing.T) {
		f := newFixture()
		f.users.On("FindProfile", mock.Anything, "u1").Return(repository.Profile{}, repository.ErrNotFound)

		_, err := f.svc.GetProfile(ctx, &authpb.GetProfileRequest{UserId: "u1"})

		requireCode(t, err, codes.NotFound)
	})
}

func TestUpdateProfile(t *testing.T) {
	ctx := context.Background()

	t.Run("partial update success", func(t *testing.T) {
		f := newFixture()
		newName := "Adrian Baru"
		f.users.On("UpdateProfile", mock.Anything, mock.MatchedBy(func(p repository.UpdateProfileParams) bool {
			return p.UserID == "u1" && p.Name != nil && *p.Name == "Adrian Baru" && p.Bio == nil
		})).Return(repository.Profile{
			ID:       "u1",
			Name:     "Adrian Baru",
			Username: "adrian",
		}, nil)

		res, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId: "u1",
			Name:   &newName,
		})
		require.NoError(t, err)
		assert.Equal(t, "Adrian Baru", res.Profile.Name)
	})

	t.Run("clear bio success", func(t *testing.T) {
		f := newFixture()
		emptyBio := ""
		f.users.On("UpdateProfile", mock.Anything, mock.MatchedBy(func(p repository.UpdateProfileParams) bool {
			return p.UserID == "u1" && p.Bio != nil && *p.Bio == ""
		})).Return(repository.Profile{
			ID:       "u1",
			Name:     "Adrian",
			Username: "adrian",
			Bio:      "",
		}, nil)

		res, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId: "u1",
			Bio:    &emptyBio,
		})
		require.NoError(t, err)
		assert.Equal(t, "", res.Profile.Bio)
	})

	t.Run("username conflict 409", func(t *testing.T) {
		f := newFixture()
		taken := "taken_user"
		f.users.On("UpdateProfile", mock.Anything, mock.Anything).Return(repository.Profile{}, repository.ErrUsernameTaken)

		_, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId:   "u1",
			Username: &taken,
		})
		requireCode(t, err, codes.AlreadyExists)
	})

	t.Run("email conflict 409", func(t *testing.T) {
		f := newFixture()
		taken := "taken@mail.com"
		f.users.On("UpdateProfile", mock.Anything, mock.Anything).Return(repository.Profile{}, repository.ErrEmailTaken)

		_, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId: "u1",
			Email:  &taken,
		})
		requireCode(t, err, codes.AlreadyExists)
	})

	t.Run("wrong old password", func(t *testing.T) {
		f := newFixture()
		oldPwd := "WrongPassword123!"
		newPwd := "NewValidPass123!"

		f.tokens.On("PasswordAttempts", mock.Anything, "u1").Return(int64(0), nil)
		f.users.On("FindByID", mock.Anything, "u1").Return(repository.Credentials{
			ID:           "u1",
			PasswordHash: "$2a$10$abcdefghijklmnopqrstuvwxyz0123456789012345678901234567890",
		}, nil)
		f.tokens.On("AddPasswordAttempt", mock.Anything, "u1", mock.Anything).Return(int64(1), nil)

		_, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId:      "u1",
			OldPassword: &oldPwd,
			NewPassword: &newPwd,
		})
		requireCode(t, err, codes.InvalidArgument)
	})

	t.Run("rate limited old password", func(t *testing.T) {
		f := newFixture()
		oldPwd := "WrongPassword123!"
		newPwd := "NewValidPass123!"

		f.tokens.On("PasswordAttempts", mock.Anything, "u1").Return(int64(5), nil)

		_, err := f.svc.UpdateProfile(ctx, &authpb.UpdateProfileRequest{
			UserId:      "u1",
			OldPassword: &oldPwd,
			NewPassword: &newPwd,
		})
		requireCode(t, err, codes.ResourceExhausted)
	})
}

func TestDeactivateAccount(t *testing.T) {
	ctx := context.Background()

	t.Run("success deactivates account and revokes tokens", func(t *testing.T) {
		f := newFixture()
		plain := "SecretPass123!"
		hashed := hash(t, plain)

		f.tokens.On("PasswordAttempts", mock.Anything, "u1").Return(int64(0), nil)
		f.users.On("FindByID", mock.Anything, "u1").Return(repository.Credentials{
			ID:           "u1",
			PasswordHash: hashed,
		}, nil)
		f.users.On("Deactivate", mock.Anything, "u1").Return(nil)
		f.tokens.On("ClearPasswordAttempts", mock.Anything, "u1").Return(nil)
		f.tokens.On("RevokeAllUserTokens", mock.Anything, "u1").Return(nil)

		res, err := f.svc.DeactivateAccount(ctx, &authpb.DeactivateAccountRequest{
			UserId:   "u1",
			Password: plain,
		})

		require.NoError(t, err)
		assert.True(t, res.Success)
		f.users.AssertExpectations(t)
		f.tokens.AssertExpectations(t)
	})

	t.Run("wrong password increments attempt", func(t *testing.T) {
		f := newFixture()
		hashed := hash(t, "RealPassword123!")

		f.tokens.On("PasswordAttempts", mock.Anything, "u1").Return(int64(0), nil)
		f.users.On("FindByID", mock.Anything, "u1").Return(repository.Credentials{
			ID:           "u1",
			PasswordHash: hashed,
		}, nil)
		f.tokens.On("AddPasswordAttempt", mock.Anything, "u1", mock.Anything).Return(int64(1), nil)

		_, err := f.svc.DeactivateAccount(ctx, &authpb.DeactivateAccountRequest{
			UserId:   "u1",
			Password: "WrongPassword!",
		})

		requireCode(t, err, codes.InvalidArgument)
	})

	t.Run("rate limited", func(t *testing.T) {
		f := newFixture()
		f.tokens.On("PasswordAttempts", mock.Anything, "u1").Return(int64(5), nil)

		_, err := f.svc.DeactivateAccount(ctx, &authpb.DeactivateAccountRequest{
			UserId:   "u1",
			Password: "AnyPassword!",
		})

		requireCode(t, err, codes.ResourceExhausted)
	})
}