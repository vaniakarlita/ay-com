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
	blockFunc            func(ctx context.Context, blockerID, blockedID string) error
	unblockFunc          func(ctx context.Context, blockerID, blockedID string) error
	checkBlockFunc       func(ctx context.Context, userA, userB string) (bool, bool, error)
	createReportFunc     func(ctx context.Context, reporterID, reportedID, reason, details string) error
	listBlockedUsersFunc func(ctx context.Context, blockerID string, limit int, cursor time.Time) ([]repository.RelationItem, error)
	getBlockedUserIDsFunc func(ctx context.Context, userID string) ([]string, error)
	getNotifSettingsFunc func(ctx context.Context, userID string) (bool, bool, bool, bool, bool, error)
	updateNotifSettingsFunc func(ctx context.Context, userID string, like, repost, follow, mention, comm bool) error
}

func (m *mockFollowRepo) Block(ctx context.Context, blockerID, blockedID string) error {
	if m.blockFunc != nil {
		return m.blockFunc(ctx, blockerID, blockedID)
	}
	return nil
}

func (m *mockFollowRepo) Unblock(ctx context.Context, blockerID, blockedID string) error {
	if m.unblockFunc != nil {
		return m.unblockFunc(ctx, blockerID, blockedID)
	}
	return nil
}

func (m *mockFollowRepo) CheckBlock(ctx context.Context, userA, userB string) (bool, bool, error) {
	if m.checkBlockFunc != nil {
		return m.checkBlockFunc(ctx, userA, userB)
	}
	return false, false, nil
}

func (m *mockFollowRepo) CreateReport(ctx context.Context, reporterID, reportedID, reason, details string) error {
	if m.createReportFunc != nil {
		return m.createReportFunc(ctx, reporterID, reportedID, reason, details)
	}
	return nil
}

func (m *mockFollowRepo) ListBlockedUsers(ctx context.Context, blockerID string, limit int, cursor time.Time) ([]repository.RelationItem, error) {
	if m.listBlockedUsersFunc != nil {
		return m.listBlockedUsersFunc(ctx, blockerID, limit, cursor)
	}
	return []repository.RelationItem{
		{UserID: "u_blocked", CreatedAt: time.Now()},
	}, nil
}

func (m *mockFollowRepo) GetBlockedUserIDs(ctx context.Context, userID string) ([]string, error) {
	if m.getBlockedUserIDsFunc != nil {
		return m.getBlockedUserIDsFunc(ctx, userID)
	}
	return []string{"u_blocked"}, nil
}

func (m *mockFollowRepo) GetNotificationSettings(ctx context.Context, userID string) (bool, bool, bool, bool, bool, error) {
	if m.getNotifSettingsFunc != nil {
		return m.getNotifSettingsFunc(ctx, userID)
	}
	return true, true, true, true, true, nil
}

func (m *mockFollowRepo) UpdateNotificationSettings(ctx context.Context, userID string, like, repost, follow, mention, comm bool) error {
	if m.updateNotifSettingsFunc != nil {
		return m.updateNotifSettingsFunc(ctx, userID, like, repost, follow, mention, comm)
	}
	return nil
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

func TestBlock_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.BlockUser(context.Background(), &userpb.BlockRequest{
		BlockerId: "u1",
		BlockedId: "u2",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
}

func TestBlock_CannotBlockSelf(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	_, err := svc.BlockUser(context.Background(), &userpb.BlockRequest{
		BlockerId: "u1",
		BlockedId: "u1",
	})
	assert.Error(t, err)
}

func TestUnblock_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.UnblockUser(context.Background(), &userpb.BlockRequest{
		BlockerId: "u1",
		BlockedId: "u2",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
}

func TestCheckBlockStatus(t *testing.T) {
	repo := &mockFollowRepo{
		checkBlockFunc: func(ctx context.Context, userA, userB string) (bool, bool, error) {
			return true, false, nil
		},
	}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.CheckBlockStatus(context.Background(), &userpb.CheckBlockRequest{
		UserA: "u1",
		UserB: "u2",
	})
	assert.NoError(t, err)
	assert.True(t, res.IsBlocking)
	assert.False(t, res.IsBlockedBy)
}

func TestReportUser_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.ReportUser(context.Background(), &userpb.ReportRequest{
		ReporterId: "u1",
		ReportedId: "u2",
		Reason:     "Spam",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
}

func TestReportUser_Validation(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	_, err := svc.ReportUser(context.Background(), &userpb.ReportRequest{
		ReporterId: "u1",
		ReportedId: "u1",
		Reason:     "Spam",
	})
	assert.Error(t, err)

	_, err = svc.ReportUser(context.Background(), &userpb.ReportRequest{
		ReporterId: "u1",
		ReportedId: "u2",
		Reason:     "InvalidReason",
	})
	assert.Error(t, err)

	_, err = svc.ReportUser(context.Background(), &userpb.ReportRequest{
		ReporterId: "u1",
		ReportedId: "u2",
		Reason:     "Other",
		Details:    "",
	})
	assert.Error(t, err)
}

func TestListBlockedUsers_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.ListBlockedUsers(context.Background(), &userpb.ListRelationRequest{
		UserId: "u1",
	})
	assert.NoError(t, err)
	assert.Len(t, res.Users, 1)
	assert.Equal(t, "u_blocked", res.Users[0].UserId)
}

func TestGetBlockedUserIDs_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	res, err := svc.GetBlockedUserIDs(context.Background(), &userpb.GetBlockedUserIDsRequest{
		UserId: "u1",
	})
	assert.NoError(t, err)
	assert.Equal(t, []string{"u_blocked"}, res.BlockedUserIds)
}

func TestNotificationSettings_Success(t *testing.T) {
	repo := &mockFollowRepo{}
	svc := service.NewUserService(repo, nil, testLogger())

	getRes, err := svc.GetNotificationSettings(context.Background(), &userpb.GetNotificationSettingsRequest{
		UserId: "u1",
	})
	assert.NoError(t, err)
	assert.True(t, getRes.NotifLike)
	assert.True(t, getRes.NotifRepost)

	upRes, err := svc.UpdateNotificationSettings(context.Background(), &userpb.UpdateNotificationSettingsRequest{
		UserId:         "u1",
		NotifLike:      false,
		NotifRepost:    true,
		NotifFollow:    true,
		NotifMention:   true,
		NotifCommunity: false,
	})
	assert.NoError(t, err)
	assert.False(t, upRes.NotifLike)
	assert.False(t, upRes.NotifCommunity)
}
