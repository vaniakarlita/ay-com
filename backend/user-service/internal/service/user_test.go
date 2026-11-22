package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/service"
	"github.com/vaniakarlita/ay-com/backend/user-service/userpb"
)

type mockFollowRepo struct {
	followFunc           func(ctx context.Context, followerID, followeeID string) error
	unfollowFunc         func(ctx context.Context, followerID, followeeID string) error
	getStatsFunc         func(ctx context.Context, targetUserID, viewerUserID string) (int32, int32, bool, error)
	listFollowingIDsFunc func(ctx context.Context, userID string, limit int) ([]string, error)
	listFollowersFunc    func(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]repository.RelationItem, error)
	listFollowingFunc    func(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]repository.RelationItem, error)
	suggestAccountsFunc  func(ctx context.Context, userID string, limit int) ([]repository.SuggestedAccount, error)
}

func (m *mockFollowRepo) Follow(ctx context.Context, followerID, followeeID string) error {
	if m.followFunc != nil {
		return m.followFunc(ctx, followerID, followeeID)
	}
	return nil
}

func (m *mockFollowRepo) Unfollow(ctx context.Context, followerID, followeeID string) error {
	if m.unfollowFunc != nil {
		return m.unfollowFunc(ctx, followerID, followeeID)
	}
	return nil
}

func (m *mockFollowRepo) GetStats(ctx context.Context, targetUserID, viewerUserID string) (int32, int32, bool, error) {
	if m.getStatsFunc != nil {
		return m.getStatsFunc(ctx, targetUserID, viewerUserID)
	}
	return 10, 5, true, nil
}

func (m *mockFollowRepo) ListFollowingIDs(ctx context.Context, userID string, limit int) ([]string, error) {
	if m.listFollowingIDsFunc != nil {
		return m.listFollowingIDsFunc(ctx, userID, limit)
	}
	return []string{"u2", "u3"}, nil
}

func (m *mockFollowRepo) ListFollowers(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]repository.RelationItem, error) {
	if m.listFollowersFunc != nil {
		return m.listFollowersFunc(ctx, targetUserID, viewerUserID, limit, cursor)
	}
	return []repository.RelationItem{
		{UserID: "u2", CreatedAt: time.Now(), IsFollowing: true},
	}, nil
}

func (m *mockFollowRepo) ListFollowing(ctx context.Context, targetUserID, viewerUserID string, limit int, cursor time.Time) ([]repository.RelationItem, error) {
	if m.listFollowingFunc != nil {
		return m.listFollowingFunc(ctx, targetUserID, viewerUserID, limit, cursor)
	}
	return []repository.RelationItem{
		{UserID: "u3", CreatedAt: time.Now(), IsFollowing: true},
	}, nil
}

func (m *mockFollowRepo) SuggestAccounts(ctx context.Context, userID string, limit int) ([]repository.SuggestedAccount, error) {
	if m.suggestAccountsFunc != nil {
		return m.suggestAccountsFunc(ctx, userID, limit)
	}
	return []repository.SuggestedAccount{
		{UserID: "u4", FollowerCount: 100},
	}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestFollow_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.Follow(context.Background(), &userpb.FollowRequest{
		FollowerId: "u1",
		FolloweeId: "u2",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
	assert.Equal(t, int32(10), res.FollowerCount)
	assert.True(t, res.IsFollowing)
}

func TestFollow_CannotFollowSelf(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	_, err := svc.Follow(context.Background(), &userpb.FollowRequest{
		FollowerId: "u1",
		FolloweeId: "u1",
	})
	assert.Error(t, err)
}

func TestUnfollow_Success(t *testing.T) {
	repo := &mockFollowRepo{
		getStatsFunc: func(ctx context.Context, targetUserID, viewerUserID string) (int32, int32, bool, error) {
			return 9, 5, false, nil
		},
	}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.Unfollow(context.Background(), &userpb.FollowRequest{
		FollowerId: "u1",
		FolloweeId: "u2",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
	assert.False(t, res.IsFollowing)
}

func TestListFollowingIDs_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.ListFollowingIDs(context.Background(), &userpb.ListFollowingIDsRequest{
		UserId: "u1",
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"u2", "u3"}, res.FollowingIds)
}

func TestSuggestAccounts_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.SuggestAccounts(context.Background(), &userpb.SuggestAccountsRequest{
		UserId: "u1",
		Limit:  3,
	})
	assert.NoError(t, err)
	assert.Len(t, res.Accounts, 1)
	assert.Equal(t, "u4", res.Accounts[0].UserId)
}

func TestFollow_RepoError(t *testing.T) {
	repo := &mockFollowRepo{
		followFunc: func(ctx context.Context, followerID, followeeID string) error {
			return errors.New("db error")
		},
	}
	svc := service.NewUserService(repo, nil, testLogger())

	_, err := svc.Follow(context.Background(), &userpb.FollowRequest{
		FollowerId: "u1",
		FolloweeId: "u2",
	})
	assert.Error(t, err)
}
