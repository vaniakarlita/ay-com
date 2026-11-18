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
	createFunc         func(ctx context.Context, t *repository.ThreadModel, hashtags []string) error
	getByIDFunc        func(ctx context.Context, id string) (*repository.ThreadModel, error)
	deleteFunc         func(ctx context.Context, id, userID string) error
	listForYouFunc     func(ctx context.Context, limit int, cursor time.Time) ([]*repository.ThreadModel, error)
	listCategoriesFunc func(ctx context.Context) ([]repository.CategoryModel, error)
}

func (m *mockRepo) Create(ctx context.Context, t *repository.ThreadModel, hashtags []string) error {
	if m.createFunc != nil {
		return m.createFunc(ctx, t, hashtags)
	}
	t.ID = "test-uuid"
	t.CreatedAt = time.Now()
	return nil
}

func (m *mockRepo) GetByID(ctx context.Context, id string) (*repository.ThreadModel, error) {
	if m.getByIDFunc != nil {
		return m.getByIDFunc(ctx, id)
	}
	return nil, nil
}

func (m *mockRepo) Delete(ctx context.Context, id, userID string) error {
	if m.deleteFunc != nil {
		return m.deleteFunc(ctx, id, userID)
	}
	return nil
}

func (m *mockRepo) ListForYou(ctx context.Context, limit int, cursor time.Time) ([]*repository.ThreadModel, error) {
	if m.listForYouFunc != nil {
		return m.listForYouFunc(ctx, limit, cursor)
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

func (m *mockRepo) ListCategories(ctx context.Context) ([]repository.CategoryModel, error) {
	if m.listCategoriesFunc != nil {
		return m.listCategoriesFunc(ctx)
	}
	return []repository.CategoryModel{
		{ID: "c1", Name: "Technology"},
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

	// Missing user_id
	_, err := svc.CreateThread(context.Background(), &threadpb.CreateThreadRequest{
		Content: "Valid content",
	})
	assert.Error(t, err)

	// Missing content
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
