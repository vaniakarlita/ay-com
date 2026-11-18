package service

import (
	"context"
	"log/slog"
	"time"

	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/parser"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/thread-service/threadpb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type ThreadService struct {
	threadpb.UnimplementedThreadServiceServer
	repo   repository.ThreadRepository
	logger *slog.Logger
}

func NewThreadService(repo repository.ThreadRepository, logger *slog.Logger) *ThreadService {
	return &ThreadService{
		repo:   repo,
		logger: logger,
	}
}

func (s *ThreadService) CreateThread(ctx context.Context, req *threadpb.CreateThreadRequest) (*threadpb.CreateThreadResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}
	if req.Content == "" {
		return nil, status.Error(codes.InvalidArgument, "content cannot be empty")
	}

	replyPerm := req.ReplyPermission
	if replyPerm == "" {
		replyPerm = "everyone"
	}

	var scheduledAt *time.Time
	if req.ScheduledAt != "" {
		if t, err := time.Parse(time.RFC3339, req.ScheduledAt); err == nil {
			scheduledAt = &t
		}
	}

	hashtags := parser.ExtractHashtags(req.Content)

	model := &repository.ThreadModel{
		UserID:           req.UserId,
		AuthorName:       req.AuthorName,
		AuthorUsername:   req.AuthorUsername,
		AuthorAvatarURL:  req.AuthorAvatarUrl,
		AuthorIsVerified: req.AuthorIsVerified,
		Content:          req.Content,
		ReplyPermission:  replyPerm,
		CommunityID:      req.CommunityId,
		IsAd:             req.IsAd,
		ScheduledAt:      scheduledAt,
		Categories:       req.Categories,
	}

	if err := s.repo.Create(ctx, model, hashtags); err != nil {
		s.logger.Error("failed to create thread", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to create thread: %v", err)
	}

	s.logger.Info("thread created", "thread_id", model.ID, "user_id", req.UserId, "is_ad", model.IsAd)

	return &threadpb.CreateThreadResponse{
		Thread: toProtoThread(model),
	}, nil
}

func (s *ThreadService) GetThread(ctx context.Context, req *threadpb.GetThreadRequest) (*threadpb.GetThreadResponse, error) {
	if req.ThreadId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id is required")
	}

	t, err := s.repo.GetByID(ctx, req.ThreadId)
	if err != nil {
		s.logger.Error("failed to get thread", "error", err, "thread_id", req.ThreadId)
		return nil, status.Errorf(codes.Internal, "failed to get thread: %v", err)
	}
	if t == nil {
		return nil, status.Error(codes.NotFound, "thread not found")
	}

	return &threadpb.GetThreadResponse{
		Thread: toProtoThread(t),
	}, nil
}

func (s *ThreadService) ListFeed(ctx context.Context, req *threadpb.ListFeedRequest) (*threadpb.ListFeedResponse, error) {
	limit := int(req.Limit)
	if limit <= 0 {
		limit = 20
	}

	var cursorTime time.Time
	if req.Cursor != "" {
		if t, err := time.Parse(time.RFC3339Nano, req.Cursor); err == nil {
			cursorTime = t
		}
	}

	threads, err := s.repo.ListForYou(ctx, limit, cursorTime)
	if err != nil {
		s.logger.Error("failed to list feed", "error", err, "user_id", req.ViewerUserId)
		return nil, status.Errorf(codes.Internal, "failed to list feed: %v", err)
	}

	protoThreads := make([]*threadpb.Thread, len(threads))
	var nextCursor string
	for i, t := range threads {
		protoThreads[i] = toProtoThread(t)
		if i == len(threads)-1 {
			nextCursor = t.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	return &threadpb.ListFeedResponse{
		Threads:    protoThreads,
		NextCursor: nextCursor,
		HasMore:    len(threads) >= limit,
	}, nil
}

func (s *ThreadService) DeleteThread(ctx context.Context, req *threadpb.DeleteThreadRequest) (*threadpb.DeleteThreadResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Delete(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Warn("failed to delete thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.PermissionDenied, "cannot delete thread: %v", err)
	}

	s.logger.Info("thread deleted", "thread_id", req.ThreadId, "user_id", req.UserId)
	return &threadpb.DeleteThreadResponse{
		Success: true,
		Message: "Thread deleted successfully",
	}, nil
}

func (s *ThreadService) GetCategories(ctx context.Context, req *threadpb.GetCategoriesRequest) (*threadpb.GetCategoriesResponse, error) {
	cats, err := s.repo.ListCategories(ctx)
	if err != nil {
		s.logger.Error("failed to list categories", "error", err)
		return nil, status.Errorf(codes.Internal, "failed to list categories: %v", err)
	}

	protoCats := make([]*threadpb.Category, len(cats))
	for i, c := range cats {
		protoCats[i] = &threadpb.Category{
			Id:   c.ID,
			Name: c.Name,
		}
	}

	return &threadpb.GetCategoriesResponse{
		Categories: protoCats,
	}, nil
}

func toProtoThread(m *repository.ThreadModel) *threadpb.Thread {
	schedStr := ""
	if m.ScheduledAt != nil && !m.ScheduledAt.IsZero() {
		schedStr = m.ScheduledAt.Format(time.RFC3339)
	}

	return &threadpb.Thread{
		Id:               m.ID,
		UserId:           m.UserID,
		AuthorName:       m.AuthorName,
		AuthorUsername:   m.AuthorUsername,
		AuthorAvatarUrl:  m.AuthorAvatarURL,
		AuthorIsVerified: m.AuthorIsVerified,
		Content:          m.Content,
		ReplyPermission:  m.ReplyPermission,
		Categories:       m.Categories,
		CommunityId:      m.CommunityID,
		CommunityName:    m.CommunityName,
		IsAd:             m.IsAd,
		ScheduledAt:      schedStr,
		CreatedAt:        m.CreatedAt.Format(time.RFC3339),
		LikeCount:        0,
		ReplyCount:       0,
		RepostCount:      0,
		BookmarkCount:    0,
		LikedByMe:        false,
		RepostedByMe:     false,
		BookmarkedByMe:   false,
	}
}
