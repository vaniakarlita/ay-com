package handler_test

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/handler"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
	"google.golang.org/grpc"
)

type mockUserClient struct {
	userpb.UserServiceClient
	checkBlockFunc     func(ctx context.Context, in *userpb.CheckBlockRequest, opts ...grpc.CallOption) (*userpb.CheckBlockResponse, error)
	getBlockedIDsFunc  func(ctx context.Context, in *userpb.GetBlockedUserIDsRequest, opts ...grpc.CallOption) (*userpb.GetBlockedUserIDsResponse, error)
	listFollowingIDsFunc func(ctx context.Context, in *userpb.ListFollowingIDsRequest, opts ...grpc.CallOption) (*userpb.ListFollowingIDsResponse, error)
	getFollowStatsFunc func(ctx context.Context, in *userpb.GetFollowStatsRequest, opts ...grpc.CallOption) (*userpb.GetFollowStatsResponse, error)
}

func (m *mockUserClient) CheckBlockStatus(ctx context.Context, in *userpb.CheckBlockRequest, opts ...grpc.CallOption) (*userpb.CheckBlockResponse, error) {
	if m.checkBlockFunc != nil {
		return m.checkBlockFunc(ctx, in, opts...)
	}
	return &userpb.CheckBlockResponse{}, nil
}

func (m *mockUserClient) GetBlockedUserIDs(ctx context.Context, in *userpb.GetBlockedUserIDsRequest, opts ...grpc.CallOption) (*userpb.GetBlockedUserIDsResponse, error) {
	if m.getBlockedIDsFunc != nil {
		return m.getBlockedIDsFunc(ctx, in, opts...)
	}
	return &userpb.GetBlockedUserIDsResponse{}, nil
}

func (m *mockUserClient) ListFollowingIDs(ctx context.Context, in *userpb.ListFollowingIDsRequest, opts ...grpc.CallOption) (*userpb.ListFollowingIDsResponse, error) {
	if m.listFollowingIDsFunc != nil {
		return m.listFollowingIDsFunc(ctx, in, opts...)
	}
	return &userpb.ListFollowingIDsResponse{}, nil
}

func (m *mockUserClient) GetFollowStats(ctx context.Context, in *userpb.GetFollowStatsRequest, opts ...grpc.CallOption) (*userpb.GetFollowStatsResponse, error) {
	if m.getFollowStatsFunc != nil {
		return m.getFollowStatsFunc(ctx, in, opts...)
	}
	return &userpb.GetFollowStatsResponse{}, nil
}

type mockAuthClient struct {
	authpb.AuthServiceClient
	getProfileFunc           func(ctx context.Context, in *authpb.GetProfileRequest, opts ...grpc.CallOption) (*authpb.GetProfileResponse, error)
	getActiveProfilesFunc   func(ctx context.Context, in *authpb.GetProfilesByIDsRequest, opts ...grpc.CallOption) (*authpb.GetProfilesByIDsResponse, error)
}

func (m *mockAuthClient) GetProfile(ctx context.Context, in *authpb.GetProfileRequest, opts ...grpc.CallOption) (*authpb.GetProfileResponse, error) {
	if m.getProfileFunc != nil {
		return m.getProfileFunc(ctx, in, opts...)
	}
	return &authpb.GetProfileResponse{UserId: in.UserId}, nil
}

func (m *mockAuthClient) GetActiveProfilesByIDs(ctx context.Context, in *authpb.GetProfilesByIDsRequest, opts ...grpc.CallOption) (*authpb.GetProfilesByIDsResponse, error) {
	if m.getActiveProfilesFunc != nil {
		return m.getActiveProfilesFunc(ctx, in, opts...)
	}
	var res []*authpb.GetProfileResponse
	for _, id := range in.UserIds {
		res = append(res, &authpb.GetProfileResponse{UserId: id})
	}
	return &authpb.GetProfilesByIDsResponse{Profiles: res}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCheckContentVisibility(t *testing.T) {
	uMock := &mockUserClient{
		checkBlockFunc: func(ctx context.Context, in *userpb.CheckBlockRequest, opts ...grpc.CallOption) (*userpb.CheckBlockResponse, error) {
			if in.UserB == "u_blocked" {
				return &userpb.CheckBlockResponse{IsBlockedBy: true}, nil
			}
			return &userpb.CheckBlockResponse{}, nil
		},
		getFollowStatsFunc: func(ctx context.Context, in *userpb.GetFollowStatsRequest, opts ...grpc.CallOption) (*userpb.GetFollowStatsResponse, error) {
			if in.TargetUserId == "u_private_following" {
				return &userpb.GetFollowStatsResponse{IsFollowing: true}, nil
			}
			return &userpb.GetFollowStatsResponse{IsFollowing: false}, nil
		},
	}

	aMock := &mockAuthClient{
		getProfileFunc: func(ctx context.Context, in *authpb.GetProfileRequest, opts ...grpc.CallOption) (*authpb.GetProfileResponse, error) {
			if in.UserId == "u_private_nofollow" || in.UserId == "u_private_following" {
				return &authpb.GetProfileResponse{UserId: in.UserId, IsPrivate: true}, nil
			}
			return &authpb.GetProfileResponse{UserId: in.UserId, IsPrivate: false}, nil
		},
	}

	h := handler.NewThreadHandler(nil, aMock, uMock, testLogger())

	// Self check
	dec, err := h.CheckContentVisibility(context.Background(), "u_me", "u_me")
	if err != nil || !dec.Allowed {
		t.Fatalf("expected allowed for self, got %v, err %v", dec.Allowed, err)
	}

	// Blocked check
	dec, err = h.CheckContentVisibility(context.Background(), "u_me", "u_blocked")
	if err != nil || dec.Allowed || dec.Reason != "blocked_by" {
		t.Fatalf("expected not allowed (blocked_by), got %v, reason %v, err %v", dec.Allowed, dec.Reason, err)
	}

	// Private without follow
	dec, err = h.CheckContentVisibility(context.Background(), "u_me", "u_private_nofollow")
	if err != nil || dec.Allowed || dec.Reason != "private" {
		t.Fatalf("expected not allowed (private), got %v, reason %v, err %v", dec.Allowed, dec.Reason, err)
	}

	// Private with follow
	dec, err = h.CheckContentVisibility(context.Background(), "u_me", "u_private_following")
	if err != nil || !dec.Allowed {
		t.Fatalf("expected allowed for private with follow, got %v, err %v", dec.Allowed, err)
	}

	// Public author
	dec, err = h.CheckContentVisibility(context.Background(), "u_me", "u_public")
	if err != nil || !dec.Allowed {
		t.Fatalf("expected allowed for public, got %v, err %v", dec.Allowed, err)
	}
}

func TestFilterVisibleAuthorIDs(t *testing.T) {
	uMock := &mockUserClient{
		getBlockedIDsFunc: func(ctx context.Context, in *userpb.GetBlockedUserIDsRequest, opts ...grpc.CallOption) (*userpb.GetBlockedUserIDsResponse, error) {
			return &userpb.GetBlockedUserIDsResponse{BlockedUserIds: []string{"u_blocked"}}, nil
		},
		listFollowingIDsFunc: func(ctx context.Context, in *userpb.ListFollowingIDsRequest, opts ...grpc.CallOption) (*userpb.ListFollowingIDsResponse, error) {
			return &userpb.ListFollowingIDsResponse{FollowingIds: []string{"u_private_following"}}, nil
		},
	}

	aMock := &mockAuthClient{
		getActiveProfilesFunc: func(ctx context.Context, in *authpb.GetProfilesByIDsRequest, opts ...grpc.CallOption) (*authpb.GetProfilesByIDsResponse, error) {
			return &authpb.GetProfilesByIDsResponse{
				Profiles: []*authpb.GetProfileResponse{
					{UserId: "u_public", IsPrivate: false},
					{UserId: "u_blocked", IsPrivate: false},
					{UserId: "u_private_following", IsPrivate: true},
					{UserId: "u_private_nofollow", IsPrivate: true},
				},
			}, nil
		},
	}

	h := handler.NewThreadHandler(nil, aMock, uMock, testLogger())

	authors := []string{"u_me", "u_public", "u_blocked", "u_private_following", "u_private_nofollow", "u_deactivated"}
	visibleMap, err := h.FilterVisibleAuthorIDs(context.Background(), "u_me", authors)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !visibleMap["u_me"] {
		t.Errorf("u_me should be visible")
	}
	if !visibleMap["u_public"] {
		t.Errorf("u_public should be visible")
	}
	if !visibleMap["u_private_following"] {
		t.Errorf("u_private_following should be visible")
	}
	if visibleMap["u_blocked"] {
		t.Errorf("u_blocked should not be visible")
	}
	if visibleMap["u_private_nofollow"] {
		t.Errorf("u_private_nofollow should not be visible")
	}
	if visibleMap["u_deactivated"] {
		t.Errorf("u_deactivated should not be visible")
	}
}
