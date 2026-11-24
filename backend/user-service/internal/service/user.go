package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/vaniakarlita/ay-com/backend/user-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/user-service/userpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type UserService struct {
	userpb.UnimplementedUserServiceServer
	repo   repository.FollowRepository
	rdb    *redis.Client
	logger *slog.Logger
}

func NewUserService(repo repository.FollowRepository, rdb *redis.Client, logger *slog.Logger) *UserService {
	return &UserService{
		repo:   repo,
		rdb:    rdb,
		logger: logger,
	}
}

func (s *UserService) Follow(ctx context.Context, req *userpb.FollowRequest) (*userpb.FollowResponse, error) {
	if req.FollowerId == "" || req.FolloweeId == "" {
		return nil, status.Error(codes.InvalidArgument, "follower_id and followee_id are required")
	}
	if req.FollowerId == req.FolloweeId {
		return nil, status.Error(codes.InvalidArgument, "cannot follow self")
	}

	if err := s.repo.Follow(ctx, req.FollowerId, req.FolloweeId); err != nil {
		s.logger.Error("follow failed", "error", err, "follower", req.FollowerId, "followee", req.FolloweeId)
		return nil, status.Errorf(codes.Internal, "follow failed: %v", err)
	}

	if s.rdb != nil {
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.FolloweeId, req.FollowerId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:", req.FolloweeId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.FollowerId, req.FolloweeId))
		s.rdb.Del(ctx, fmt.Sprintf("suggest:%s", req.FollowerId))
	}

	followers, following, isFollowing, err := s.repo.GetStats(ctx, req.FolloweeId, req.FollowerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get stats failed: %v", err)
	}

	return &userpb.FollowResponse{
		Success:        true,
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
	}, nil
}

func (s *UserService) Unfollow(ctx context.Context, req *userpb.FollowRequest) (*userpb.FollowResponse, error) {
	if req.FollowerId == "" || req.FolloweeId == "" {
		return nil, status.Error(codes.InvalidArgument, "follower_id and followee_id are required")
	}

	if err := s.repo.Unfollow(ctx, req.FollowerId, req.FolloweeId); err != nil {
		s.logger.Error("unfollow failed", "error", err, "follower", req.FollowerId, "followee", req.FolloweeId)
		return nil, status.Errorf(codes.Internal, "unfollow failed: %v", err)
	}

	if s.rdb != nil {
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.FolloweeId, req.FollowerId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:", req.FolloweeId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.FollowerId, req.FolloweeId))
		s.rdb.Del(ctx, fmt.Sprintf("suggest:%s", req.FollowerId))
	}

	followers, following, isFollowing, err := s.repo.GetStats(ctx, req.FolloweeId, req.FollowerId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get stats failed: %v", err)
	}

	return &userpb.FollowResponse{
		Success:        true,
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
	}, nil
}

func (s *UserService) GetFollowStats(ctx context.Context, req *userpb.GetFollowStatsRequest) (*userpb.GetFollowStatsResponse, error) {
	if req.TargetUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "target_user_id is required")
	}

	cacheKey := fmt.Sprintf("stats:%s:%s", req.TargetUserId, req.ViewerUserId)
	if s.rdb != nil {
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached userpb.GetFollowStatsResponse
			if json.Unmarshal([]byte(val), &cached) == nil {
				return &cached, nil
			}
		}
	}

	followers, following, isFollowing, err := s.repo.GetStats(ctx, req.TargetUserId, req.ViewerUserId)
	if err != nil {
		s.logger.Error("get follow stats failed", "error", err, "target", req.TargetUserId)
		return nil, status.Errorf(codes.Internal, "get stats failed: %v", err)
	}

	res := &userpb.GetFollowStatsResponse{
		UserId:         req.TargetUserId,
		FollowerCount:  followers,
		FollowingCount: following,
		IsFollowing:    isFollowing,
	}

	if s.rdb != nil {
		if data, err := json.Marshal(res); err == nil {
			s.rdb.Set(ctx, cacheKey, data, 1*time.Minute)
		}
	}

	return res, nil
}

func (s *UserService) ListFollowingIDs(ctx context.Context, req *userpb.ListFollowingIDsRequest) (*userpb.ListFollowingIDsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	ids, err := s.repo.ListFollowingIDs(ctx, req.UserId, int(req.Limit))
	if err != nil {
		s.logger.Error("list following ids failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "list following ids failed: %v", err)
	}

	return &userpb.ListFollowingIDsResponse{FollowingIds: ids}, nil
}

func (s *UserService) ListFollowers(ctx context.Context, req *userpb.ListRelationRequest) (*userpb.ListRelationResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var cursorTime time.Time
	if req.Cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.Cursor); err == nil {
			cursorTime = t
		}
	}

	items, err := s.repo.ListFollowers(ctx, req.UserId, req.ViewerUserId, int(req.Limit), cursorTime)
	if err != nil {
		s.logger.Error("list followers failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "list followers failed: %v", err)
	}

	var pbItems []*userpb.UserRelationItem
	var nextCursor string
	for i, it := range items {
		pbItems = append(pbItems, &userpb.UserRelationItem{
			UserId:      it.UserID,
			CreatedAt:   it.CreatedAt.Format(time.RFC3339Nano),
			IsFollowing: it.IsFollowing,
		})
		if i == len(items)-1 {
			nextCursor = it.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	return &userpb.ListRelationResponse{
		Users:      pbItems,
		NextCursor: nextCursor,
		HasMore:    len(items) >= limit,
	}, nil
}

func (s *UserService) ListFollowing(ctx context.Context, req *userpb.ListRelationRequest) (*userpb.ListRelationResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var cursorTime time.Time
	if req.Cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.Cursor); err == nil {
			cursorTime = t
		}
	}

	items, err := s.repo.ListFollowing(ctx, req.UserId, req.ViewerUserId, int(req.Limit), cursorTime)
	if err != nil {
		s.logger.Error("list following failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "list following failed: %v", err)
	}

	var pbItems []*userpb.UserRelationItem
	var nextCursor string
	for i, it := range items {
		pbItems = append(pbItems, &userpb.UserRelationItem{
			UserId:      it.UserID,
			CreatedAt:   it.CreatedAt.Format(time.RFC3339Nano),
			IsFollowing: it.IsFollowing,
		})
		if i == len(items)-1 {
			nextCursor = it.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	return &userpb.ListRelationResponse{
		Users:      pbItems,
		NextCursor: nextCursor,
		HasMore:    len(items) >= limit,
	}, nil
}

func (s *UserService) SuggestAccounts(ctx context.Context, req *userpb.SuggestAccountsRequest) (*userpb.SuggestAccountsResponse, error) {
	cacheKey := fmt.Sprintf("suggest:%s", req.UserId)
	if s.rdb != nil {
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached userpb.SuggestAccountsResponse
			if json.Unmarshal([]byte(val), &cached) == nil {
				return &cached, nil
			}
		}
	}

	list, err := s.repo.SuggestAccounts(ctx, req.UserId, int(req.Limit))
	if err != nil {
		s.logger.Error("suggest accounts failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "suggest accounts failed: %v", err)
	}

	var pbList []*userpb.SuggestedAccountItem
	for _, it := range list {
		pbList = append(pbList, &userpb.SuggestedAccountItem{
			UserId:        it.UserID,
			FollowerCount: it.FollowerCount,
		})
	}

	res := &userpb.SuggestAccountsResponse{Accounts: pbList}
	if s.rdb != nil {
		if data, err := json.Marshal(res); err == nil {
			s.rdb.Set(ctx, cacheKey, data, 2*time.Minute)
		}
	}

	return res, nil
}

var allowedReportReasons = map[string]bool{
	"Spam":                    true,
	"Harassment or bullying":  true,
	"Hate speech":             true,
	"Misinformation":          true,
	"Other":                   true,
}

func (s *UserService) BlockUser(ctx context.Context, req *userpb.BlockRequest) (*userpb.BlockResponse, error) {
	if req.BlockerId == "" || req.BlockedId == "" {
		return nil, status.Error(codes.InvalidArgument, "blocker_id and blocked_id are required")
	}
	if req.BlockerId == req.BlockedId {
		return nil, status.Error(codes.InvalidArgument, "cannot block self")
	}

	if err := s.repo.Block(ctx, req.BlockerId, req.BlockedId); err != nil {
		s.logger.Error("block user failed", "error", err, "blocker", req.BlockerId, "blocked", req.BlockedId)
		return nil, status.Errorf(codes.Internal, "block failed: %v", err)
	}

	if s.rdb != nil {
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.BlockedId, req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.BlockerId, req.BlockedId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockedId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:", req.BlockedId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:", req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("suggest:%s", req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("suggest:%s", req.BlockedId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockedId))
	}

	return &userpb.BlockResponse{
		Success: true,
		Message: "User blocked successfully",
	}, nil
}

func (s *UserService) UnblockUser(ctx context.Context, req *userpb.BlockRequest) (*userpb.BlockResponse, error) {
	if req.BlockerId == "" || req.BlockedId == "" {
		return nil, status.Error(codes.InvalidArgument, "blocker_id and blocked_id are required")
	}
	if req.BlockerId == req.BlockedId {
		return nil, status.Error(codes.InvalidArgument, "cannot unblock self")
	}

	if err := s.repo.Unblock(ctx, req.BlockerId, req.BlockedId); err != nil {
		s.logger.Error("unblock user failed", "error", err, "blocker", req.BlockerId, "blocked", req.BlockedId)
		return nil, status.Errorf(codes.Internal, "unblock failed: %v", err)
	}

	if s.rdb != nil {
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.BlockedId, req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("stats:%s:%s", req.BlockerId, req.BlockedId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockerId))
		s.rdb.Del(ctx, fmt.Sprintf("blocked_ids:%s", req.BlockedId))
	}

	return &userpb.BlockResponse{
		Success: true,
		Message: "User unblocked successfully",
	}, nil
}

func (s *UserService) CheckBlockStatus(ctx context.Context, req *userpb.CheckBlockRequest) (*userpb.CheckBlockResponse, error) {
	if req.UserA == "" || req.UserB == "" || req.UserA == req.UserB {
		return &userpb.CheckBlockResponse{IsBlocking: false, IsBlockedBy: false}, nil
	}

	isBlocking, isBlockedBy, err := s.repo.CheckBlock(ctx, req.UserA, req.UserB)
	if err != nil {
		s.logger.Error("check block status failed", "error", err, "user_a", req.UserA, "user_b", req.UserB)
		return nil, status.Errorf(codes.Internal, "check block failed: %v", err)
	}

	return &userpb.CheckBlockResponse{
		IsBlocking:  isBlocking,
		IsBlockedBy: isBlockedBy,
	}, nil
}

func (s *UserService) ReportUser(ctx context.Context, req *userpb.ReportRequest) (*userpb.ReportResponse, error) {
	if req.ReporterId == "" || req.ReportedId == "" {
		return nil, status.Error(codes.InvalidArgument, "reporter_id and reported_id are required")
	}
	if req.ReporterId == req.ReportedId {
		return nil, status.Error(codes.InvalidArgument, "cannot report self")
	}
	if !allowedReportReasons[req.Reason] {
		return nil, status.Error(codes.InvalidArgument, "invalid report reason")
	}
	if req.Reason == "Other" && req.Details == "" {
		return nil, status.Error(codes.InvalidArgument, "details are required when reason is Other")
	}

	if err := s.repo.CreateReport(ctx, req.ReporterId, req.ReportedId, req.Reason, req.Details); err != nil {
		s.logger.Error("report user failed", "error", err, "reporter", req.ReporterId, "reported", req.ReportedId)
		return nil, status.Errorf(codes.Internal, "report failed: %v", err)
	}

	return &userpb.ReportResponse{
		Success: true,
		Message: "Report submitted successfully",
	}, nil
}

func (s *UserService) ListBlockedUsers(ctx context.Context, req *userpb.ListRelationRequest) (*userpb.ListRelationResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	var cursorTime time.Time
	if req.Cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.Cursor); err == nil {
			cursorTime = t
		}
	}

	items, err := s.repo.ListBlockedUsers(ctx, req.UserId, int(req.Limit), cursorTime)
	if err != nil {
		s.logger.Error("list blocked users failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "list blocked users failed: %v", err)
	}

	var pbItems []*userpb.UserRelationItem
	var nextCursor string
	for i, it := range items {
		pbItems = append(pbItems, &userpb.UserRelationItem{
			UserId:      it.UserID,
			CreatedAt:   it.CreatedAt.Format(time.RFC3339Nano),
			IsFollowing: false,
		})
		if i == len(items)-1 {
			nextCursor = it.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	return &userpb.ListRelationResponse{
		Users:      pbItems,
		NextCursor: nextCursor,
		HasMore:    len(items) >= limit,
	}, nil
}

func (s *UserService) GetBlockedUserIDs(ctx context.Context, req *userpb.GetBlockedUserIDsRequest) (*userpb.GetBlockedUserIDsResponse, error) {
	if req.UserId == "" {
		return &userpb.GetBlockedUserIDsResponse{BlockedUserIds: []string{}}, nil
	}

	cacheKey := fmt.Sprintf("blocked_ids:%s", req.UserId)
	if s.rdb != nil {
		if val, err := s.rdb.Get(ctx, cacheKey).Result(); err == nil {
			var cached []string
			if json.Unmarshal([]byte(val), &cached) == nil {
				return &userpb.GetBlockedUserIDsResponse{BlockedUserIds: cached}, nil
			}
		}
	}

	ids, err := s.repo.GetBlockedUserIDs(ctx, req.UserId)
	if err != nil {
		s.logger.Error("get blocked user ids failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "get blocked user ids failed: %v", err)
	}

	if s.rdb != nil {
		if data, err := json.Marshal(ids); err == nil {
			s.rdb.Set(ctx, cacheKey, data, 1*time.Minute)
		}
	}

	return &userpb.GetBlockedUserIDsResponse{BlockedUserIds: ids}, nil
}

func (s *UserService) GetNotificationSettings(ctx context.Context, req *userpb.GetNotificationSettingsRequest) (*userpb.NotificationSettingsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	like, repost, follow, mention, comm, err := s.repo.GetNotificationSettings(ctx, req.UserId)
	if err != nil {
		s.logger.Error("get notification settings failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "get notification settings failed: %v", err)
	}

	return &userpb.NotificationSettingsResponse{
		NotifLike:      like,
		NotifRepost:    repost,
		NotifFollow:    follow,
		NotifMention:   mention,
		NotifCommunity: comm,
	}, nil
}

func (s *UserService) UpdateNotificationSettings(ctx context.Context, req *userpb.UpdateNotificationSettingsRequest) (*userpb.NotificationSettingsResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

	err := s.repo.UpdateNotificationSettings(ctx, req.UserId, req.NotifLike, req.NotifRepost, req.NotifFollow, req.NotifMention, req.NotifCommunity)
	if err != nil {
		s.logger.Error("update notification settings failed", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "update notification settings failed: %v", err)
	}

	return &userpb.NotificationSettingsResponse{
		NotifLike:      req.NotifLike,
		NotifRepost:    req.NotifRepost,
		NotifFollow:    req.NotifFollow,
		NotifMention:   req.NotifMention,
		NotifCommunity: req.NotifCommunity,
	}, nil
}
