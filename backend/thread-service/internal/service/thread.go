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

	var parentID *string
	if req.ParentId != "" {
		parentID = &req.ParentId
	}

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
		ParentID:         parentID,
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

	t, err := s.repo.GetByID(ctx, req.ThreadId, req.ViewerUserId)
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

func (s *ThreadService) ListReplies(ctx context.Context, req *threadpb.ListRepliesRequest) (*threadpb.ListRepliesResponse, error) {
	if req.ParentId == "" {
		return nil, status.Error(codes.InvalidArgument, "parent_id is required")
	}

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

	replies, err := s.repo.ListReplies(ctx, req.ParentId, req.ViewerUserId, limit, cursorTime)
	if err != nil {
		s.logger.Error("failed to list replies", "error", err, "parent_id", req.ParentId)
		return nil, status.Errorf(codes.Internal, "failed to list replies: %v", err)
	}

	protoReplies := make([]*threadpb.Thread, len(replies))
	var nextCursor string
	for i, t := range replies {
		protoReplies[i] = toProtoThread(t)
		if i == len(replies)-1 {
			nextCursor = t.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	return &threadpb.ListRepliesResponse{
		Replies:    protoReplies,
		NextCursor: nextCursor,
		HasMore:    len(replies) >= limit,
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

	threads, err := s.repo.ListForYou(ctx, req.ViewerUserId, limit, cursorTime)
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

func (s *ThreadService) ListUserThreads(ctx context.Context, req *threadpb.ListUserThreadsRequest) (*threadpb.ListFeedResponse, error) {
	if req.AuthorUserId == "" {
		return nil, status.Error(codes.InvalidArgument, "author_user_id is required")
	}

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

	threads, err := s.repo.ListUserThreads(ctx, req.AuthorUserId, req.ViewerUserId, limit, cursorTime)
	if err != nil {
		s.logger.Error("failed to list user threads", "error", err, "author_user_id", req.AuthorUserId)
		return nil, status.Errorf(codes.Internal, "failed to list user threads: %v", err)
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

func (s *ThreadService) ListFollowingFeed(ctx context.Context, req *threadpb.ListFollowingFeedRequest) (*threadpb.ListFeedResponse, error) {
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

	threads, err := s.repo.ListFollowing(ctx, req.ViewerUserId, req.AuthorUserIds, limit, cursorTime)
	if err != nil {
		s.logger.Error("failed to list following feed", "error", err, "user_id", req.ViewerUserId)
		return nil, status.Errorf(codes.Internal, "failed to list following feed: %v", err)
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

func (s *ThreadService) LikeThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Like(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to like thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to like thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) UnlikeThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Unlike(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to unlike thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to unlike thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) RepostThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Repost(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to repost thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to repost thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) UnrepostThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Unrepost(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to unrepost thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to unrepost thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) BookmarkThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Bookmark(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to bookmark thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to bookmark thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) UnbookmarkThread(ctx context.Context, req *threadpb.InteractionRequest) (*threadpb.InteractionResponse, error) {
	if req.ThreadId == "" || req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "thread_id and user_id are required")
	}

	if err := s.repo.Unbookmark(ctx, req.ThreadId, req.UserId); err != nil {
		s.logger.Error("failed to unbookmark thread", "error", err, "thread_id", req.ThreadId, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to unbookmark thread: %v", err)
	}

	likes, reposts, bookmarks, isLiked, isReposted, isBookmarked, err := s.repo.GetCountsAndStatus(ctx, req.ThreadId, req.UserId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get counts: %v", err)
	}

	return &threadpb.InteractionResponse{
		Success:        true,
		LikeCount:      likes,
		RepostCount:    reposts,
		BookmarkCount:  bookmarks,
		LikedByMe:      isLiked,
		RepostedByMe:   isReposted,
		BookmarkedByMe: isBookmarked,
	}, nil
}

func (s *ThreadService) ListBookmarks(ctx context.Context, req *threadpb.ListBookmarksRequest) (*threadpb.ListBookmarksResponse, error) {
	if req.UserId == "" {
		return nil, status.Error(codes.InvalidArgument, "user_id is required")
	}

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

	threads, err := s.repo.ListBookmarks(ctx, req.UserId, req.Query, limit, cursorTime)
	if err != nil {
		s.logger.Error("failed to list bookmarks", "error", err, "user_id", req.UserId)
		return nil, status.Errorf(codes.Internal, "failed to list bookmarks: %v", err)
	}

	protoThreads := make([]*threadpb.Thread, len(threads))
	var nextCursor string
	for i, t := range threads {
		protoThreads[i] = toProtoThread(t)
		if i == len(threads)-1 {
			nextCursor = t.CreatedAt.Format(time.RFC3339Nano)
		}
	}

	return &threadpb.ListBookmarksResponse{
		Threads:    protoThreads,
		NextCursor: nextCursor,
		HasMore:    len(threads) >= limit,
	}, nil
}

func toProtoThread(m *repository.ThreadModel) *threadpb.Thread {
	schedStr := ""
	if m.ScheduledAt != nil && !m.ScheduledAt.IsZero() {
		schedStr = m.ScheduledAt.Format(time.RFC3339)
	}

	parentIDStr := ""
	if m.ParentID != nil {
		parentIDStr = *m.ParentID
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
		LikeCount:        m.LikeCount,
		ReplyCount:       m.ReplyCount,
		RepostCount:      m.RepostCount,
		BookmarkCount:    m.BookmarkCount,
		LikedByMe:        m.LikedByMe,
		RepostedByMe:     m.RepostedByMe,
		BookmarkedByMe:   m.BookmarkedByMe,
		RepostedByName:   m.RepostedByName,
		ParentId:         parentIDStr,
	}
}
