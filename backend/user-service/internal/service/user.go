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
