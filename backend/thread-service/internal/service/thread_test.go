package service_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/repository"
	"github.com/vaniakarlita/ay-com/backend/thread-service/internal/service"
	"github.com/vaniakarlita/ay-com/backend/thread-service/threadpb"
)

type mockRepo struct {
	createFunc             func(ctx context.Context, t *repository.ThreadModel, hashtags []string) error
	getByIDFunc            func(ctx context.Context, id, viewerUserID string) (*repository.ThreadModel, error)
	deleteFunc             func(ctx context.Context, id, userID string) error
	listForYouFunc         func(ctx context.Context, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error)
	listCategoriesFunc     func(ctx context.Context) ([]repository.CategoryModel, error)
	likeFunc               func(ctx context.Context, threadID, userID string) error
	unlikeFunc             func(ctx context.Context, threadID, userID string) error
	repostFunc             func(ctx context.Context, threadID, userID string) error
	unrepostFunc           func(ctx context.Context, threadID, userID string) error
	bookmarkFunc           func(ctx context.Context, threadID, userID string) error
	unbookmarkFunc         func(ctx context.Context, threadID, userID string) error
	getCountsAndStatusFunc func(ctx context.Context, threadID, userID string) (int32, int32, int32, bool, bool, bool, error)
	listBookmarksFunc      func(ctx context.Context, userID, query string, limit int, cursor time.Time) ([]*repository.ThreadModel, error)
	listRepliesFunc        func(ctx context.Context, parentID, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error)
	listUserThreadsFunc    func(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error)
}

func (m *mockRepo) Create(ctx context.Context, t *repository.ThreadModel, hashtags []string) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, t, hashtags)
	}
	t.ID = "test-uuid"
	t.CreatedAt = time.Now()
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id, viewerUserID string) (*repository.ThreadModel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id, viewerUserID)
	}
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}

func (m *mockRepo) ListForYou(ctx context.Context, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	if m.listForYouFunc != nil {
		return m.listForYouFunc(ctx, viewerUserID, limit, cursor)
	}
	return []*repository.ThreadModel{
		{
			ID:             "1",
			UserID:         "user-1",
			AuthorName:     "Adrian",
			AuthorUsername: "adrian",
			Content:        "First thread",
			CreatedAt:      time.Now(),
		},
	}, nil
}

func (m *mockRepo) ListFollowing(ctx context.Context, viewerUserID string, authorUserIDs []string, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	return []*repository.ThreadModel{
		{
			ID:             "f1",
			UserID:         "user-2",
			AuthorName:     "John",
			AuthorUsername: "john",
			Content:        "Following thread",
			CreatedAt:      time.Now(),
		},
	}, nil
}

func (m *mockRepo) ListReplies(ctx context.Context, parentID, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	if m.listRepliesFunc != nil {
		return m.listRepliesFunc(ctx, parentID, viewerUserID, limit, cursor)
	}
	return []*repository.ThreadModel{
		{
			ID:             "r1",
			UserID:         "user-3",
			AuthorName:     "Alice",
			AuthorUsername: "alice",
			Content:        "Nice reply!",
			CreatedAt:      time.Now(),
		},
	}, nil
}

func (m *mockRepo) ListUserThreads(ctx context.Context, authorUserID, viewerUserID string, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	if m.listUserThreadsFunc != nil {
		return m.listUserThreadsFunc(ctx, authorUserID, viewerUserID, limit, cursor)
	}
	return []*repository.ThreadModel{
		{
			ID:             "u1",
			UserID:         authorUserID,
			AuthorName:     "Adrian",
			AuthorUsername: "adrian",
			Content:        "My original thread",
			CreatedAt:      time.Now(),
		},
	}, nil
}

func (m *mockRepo) ListCategories(ctx context.Context) ([]repository.CategoryModel, error) {
	if m.listCategoriesFunc != nil {
		return m.listCategoriesFunc(ctx)
	}
	return []repository.CategoryModel{
		{ID: "c1", Name: "Technology"},
	}, nil
}

func (m *mockRepo) Like(ctx context.Context, threadID, userID string) error {
	if m.likeFunc != nil {
		return m.likeFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) Unlike(ctx context.Context, threadID, userID string) error {
	if m.unlikeFunc != nil {
		return m.unlikeFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) Repost(ctx context.Context, threadID, userID string) error {
	if m.repostFunc != nil {
		return m.repostFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) Unrepost(ctx context.Context, threadID, userID string) error {
	if m.unrepostFunc != nil {
		return m.unrepostFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) Bookmark(ctx context.Context, threadID, userID string) error {
	if m.bookmarkFunc != nil {
		return m.bookmarkFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) Unbookmark(ctx context.Context, threadID, userID string) error {
	if m.unbookmarkFunc != nil {
		return m.unbookmarkFunc(ctx, threadID, userID)
	}
	return nil
}

func (m *mockRepo) GetCountsAndStatus(ctx context.Context, threadID, userID string) (int32, int32, int32, bool, bool, bool, error) {
	if m.getCountsAndStatusFunc != nil {
		return m.getCountsAndStatusFunc(ctx, threadID, userID)
	}
	return 1, 0, 1, true, false, true, nil
}

func (m *mockRepo) ListBookmarks(ctx context.Context, userID, query string, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	if m.listBookmarksFunc != nil {
		return m.listBookmarksFunc(ctx, userID, query, limit, cursor)
	}
	return []*repository.ThreadModel{
		{
			ID:             "b1",
			UserID:         "user-1",
			AuthorName:     "Adrian",
			AuthorUsername: "adrian",
			Content:        "Bookmarked thread",
			CreatedAt:      time.Now(),
			BookmarkedByMe: true,
		},
	}, nil
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}

func TestCreateThread_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	req := &threadpb.CreateThreadRequest{
		UserId:         "user-1",
		AuthorName:     "Adrian",
		AuthorUsername: "adrian",
		Content:        "Hello #testing AY.com!",
		Categories:     []string{"Technology"},
	}

	res, err := svc.CreateThread(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "user-1", res.Thread.UserId)
	assert.Equal(t, "Hello #testing AY.com!", res.Thread.Content)
	assert.Contains(t, res.Thread.Categories, "Technology")
}

func TestCreateThread_Validation(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	_, err := svc.CreateThread(context.Background(), &threadpb.CreateThreadRequest{
		Content: "Valid content",
	})
	assert.Error(t, err)

	_, err = svc.CreateThread(context.Background(), &threadpb.CreateThreadRequest{
		UserId: "user-1",
	})
	assert.Error(t, err)
}

func TestListFeed_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	res, err := svc.ListFeed(context.Background(), &threadpb.ListFeedRequest{
		ViewerUserId: "user-1",
		Limit:        10,
	})
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Len(t, res.Threads, 1)
	assert.Equal(t, "First thread", res.Threads[0].Content)
}

func TestDeleteThread_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	res, err := svc.DeleteThread(context.Background(), &threadpb.DeleteThreadRequest{
		ThreadId: "thread-1",
		UserId:   "user-1",
	})
	assert.NoError(t, err)
	assert.True(t, res.Success)
}

func TestDeleteThread_Error(t *testing.T) {
	repo := &mockRepo{
		deleteFunc: func(ctx context.Context, id, userID string) error {
			return errors.New("unauthorized")
		},
	}
	svc := service.NewThreadService(repo, testLogger())

	_, err := svc.DeleteThread(context.Background(), &threadpb.DeleteThreadRequest{
		ThreadId: "thread-1",
		UserId:   "user-2",
	})
	assert.Error(t, err)
}

func TestLikeUnlike_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	likeRes, err := svc.LikeThread(context.Background(), &threadpb.InteractionRequest{
		ThreadId: "thread-1",
		UserId:   "user-1",
	})
	assert.NoError(t, err)
	assert.True(t, likeRes.Success)
	assert.True(t, likeRes.LikedByMe)

	unlikeRes, err := svc.UnlikeThread(context.Background(), &threadpb.InteractionRequest{
		ThreadId: "thread-1",
		UserId:   "user-1",
	})
	assert.NoError(t, err)
	assert.True(t, unlikeRes.Success)
}

func TestBookmarks_Success(t *testing.T) {
	repo := &mockRepo{}
	svc := service.NewThreadService(repo, testLogger())

	bmRes, err := svc.BookmarkThread(context.Background(), &threadpb.InteractionRequest{
		ThreadId: "thread-1",
		UserId:   "user-1",
	})
	assert.NoError(t, err)
	assert.True(t, bmRes.Success)

	listRes, err := svc.ListBookmarks(context.Background(), &threadpb.ListBookmarksRequest{
		UserId: "user-1",
	})
	assert.NoError(t, err)
	assert.Len(t, listRes.Threads, 1)
	assert.Equal(t, "Bookmarked thread", listRes.Threads[0].Content)
}
