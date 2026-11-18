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