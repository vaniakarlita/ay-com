package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
	"github.com/vaniakarlita/ay-com/backend/gateway/threadpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
)

type ThreadHandler struct {
	threadClient threadpb.ThreadServiceClient
	authClient   authpb.AuthServiceClient
	userClient   userpb.UserServiceClient
	log          *slog.Logger
}

func NewThreadHandler(
	threadClient threadpb.ThreadServiceClient,
	authClient authpb.AuthServiceClient,
	userClient userpb.UserServiceClient,
	log *slog.Logger,
) *ThreadHandler {
	return &ThreadHandler{
		threadClient: threadClient,
		authClient:   authClient,
		userClient:   userClient,
		log:          log,
	}
}

type createThreadPayload struct {
	Content         string   `json:"content"`
	ReplyPermission string   `json:"replyPermission"`
	Categories      []string `json:"categories"`
	CommunityID     string   `json:"communityId"`
	ScheduledAt     string   `json:"scheduledAt"`
	ParentID        string   `json:"parentId"`
}

func (h *ThreadHandler) Routes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.HandleFunc("GET /api/categories", h.GetCategories)
	mux.Handle("POST /api/threads", auth(http.HandlerFunc(h.CreateThread)))
	mux.Handle("GET /api/threads/feed", auth(http.HandlerFunc(h.ListFeed)))
	mux.Handle("GET /api/threads/{id}", auth(http.HandlerFunc(h.GetThread)))
	mux.Handle("GET /api/threads/{id}/replies", auth(http.HandlerFunc(h.ListReplies)))
	mux.Handle("DELETE /api/threads/{id}", auth(http.HandlerFunc(h.DeleteThread)))

	mux.Handle("POST /api/threads/{id}/like", auth(http.HandlerFunc(h.LikeThread)))
	mux.Handle("DELETE /api/threads/{id}/like", auth(http.HandlerFunc(h.UnlikeThread)))
	mux.Handle("POST /api/threads/{id}/repost", auth(http.HandlerFunc(h.RepostThread)))
	mux.Handle("DELETE /api/threads/{id}/repost", auth(http.HandlerFunc(h.UnrepostThread)))
	mux.Handle("POST /api/threads/{id}/bookmark", auth(http.HandlerFunc(h.BookmarkThread)))
	mux.Handle("DELETE /api/threads/{id}/bookmark", auth(http.HandlerFunc(h.UnbookmarkThread)))
	mux.Handle("GET /api/bookmarks", auth(http.HandlerFunc(h.ListBookmarks)))
}

func (h *ThreadHandler) CreateThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	if userID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "Unauthorized"})
		return
	}

	var body createThreadPayload
	if !decode(w, r, &body) {
		return
	}

	if body.Content == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Content cannot be empty"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	prof, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: userID})
	authorName := "User"
	authorUsername := "user"
	authorAvatar := ""
	if err == nil && prof != nil {
		authorName = prof.Name
		authorUsername = prof.Username
		authorAvatar = prof.ProfilePictureUrl
	}

	res, err := h.threadClient.CreateThread(ctx, &threadpb.CreateThreadRequest{
		UserId:           userID,
		AuthorName:       authorName,
		AuthorUsername:   authorUsername,
		AuthorAvatarUrl:  authorAvatar,
		AuthorIsVerified: false,
		Content:          body.Content,
		ReplyPermission:  body.ReplyPermission,
		Categories:       body.Categories,
		CommunityId:      body.CommunityID,
		ScheduledAt:      body.ScheduledAt,
		ParentId:         body.ParentID,
	})
	if err != nil {
		h.log.Error("failed to create thread", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to create thread"})
		return
	}

	writeJSON(w, http.StatusCreated, res.Thread)
}

func (h *ThreadHandler) ListFeed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	feedType := r.URL.Query().Get("type")
	limitStr := r.URL.Query().Get("limit")
	cursor := r.URL.Query().Get("cursor")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	if feedType == "following" {
		followRes, err := h.userClient.ListFollowingIDs(ctx, &userpb.ListFollowingIDsRequest{
			UserId: userID,
			Limit:  500,
		})
		if err != nil || len(followRes.FollowingIds) == 0 {
			writeJSON(w, http.StatusOK, map[string]any{
				"threads":    []any{},
				"nextCursor": "",
				"hasMore":    false,
			})
			return
		}

		res, err := h.threadClient.ListFollowingFeed(ctx, &threadpb.ListFollowingFeedRequest{
			ViewerUserId:  userID,
			AuthorUserIds: followRes.FollowingIds,
			Limit:         int32(limit),
			Cursor:        cursor,
		})
		if err != nil {
			h.log.Error("failed to list following feed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch following feed"})
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"threads":    res.Threads,
			"nextCursor": res.NextCursor,
			"hasMore":    res.HasMore,
		})
		return
	}

	res, err := h.threadClient.ListFeed(ctx, &threadpb.ListFeedRequest{
		ViewerUserId: userID,
		FeedType:     "for-you",
		Limit:        int32(limit),
		Cursor:       cursor,
	})
	if err != nil {
		h.log.Error("failed to list feed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch feed"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"threads":    res.Threads,
		"nextCursor": res.NextCursor,
		"hasMore":    res.HasMore,
	})
}

func (h *ThreadHandler) DeleteThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")
	if threadID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Thread ID required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.DeleteThread(ctx, &threadpb.DeleteThreadRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to delete thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "Cannot delete thread"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": res.Success,
		"message": res.Message,
	})
}

func (h *ThreadHandler) GetCategories(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.GetCategories(ctx, &threadpb.GetCategoriesRequest{})
	if err != nil {
		h.log.Error("failed to get categories", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch categories"})
		return
	}

	writeJSON(w, http.StatusOK, res.Categories)
}

func (h *ThreadHandler) LikeThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.LikeThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to like thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to like thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) UnlikeThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.UnlikeThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to unlike thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to unlike thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) RepostThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.RepostThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to repost thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to repost thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) UnrepostThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.UnrepostThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to unrepost thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to unrepost thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) BookmarkThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.BookmarkThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to bookmark thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to bookmark thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) UnbookmarkThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.UnbookmarkThread(ctx, &threadpb.InteractionRequest{
		ThreadId: threadID,
		UserId:   userID,
	})
	if err != nil {
		h.log.Error("failed to unbookmark thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to unbookmark thread"})
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (h *ThreadHandler) ListBookmarks(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	query := r.URL.Query().Get("q")
	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.ListBookmarks(ctx, &threadpb.ListBookmarksRequest{
		UserId: userID,
		Query:  query,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("failed to list bookmarks", "error", err, "user_id", userID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch bookmarks"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"threads":    res.Threads,
		"nextCursor": res.NextCursor,
		"hasMore":    res.HasMore,
	})
}

func (h *ThreadHandler) GetThread(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	threadID := r.PathValue("id")
	if threadID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Thread ID required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.GetThread(ctx, &threadpb.GetThreadRequest{
		ThreadId:     threadID,
		ViewerUserId: userID,
	})
	if err != nil {
		h.log.Error("failed to get thread", "error", err, "thread_id", threadID)
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "Thread not found"})
		return
	}

	writeJSON(w, http.StatusOK, res.Thread)
}

func (h *ThreadHandler) ListReplies(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserID(r.Context())
	parentID := r.PathValue("id")
	if parentID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Parent Thread ID required"})
		return
	}

	cursor := r.URL.Query().Get("cursor")
	limitStr := r.URL.Query().Get("limit")

	limit := 20
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 50 {
			limit = l
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.threadClient.ListReplies(ctx, &threadpb.ListRepliesRequest{
		ParentId:     parentID,
		ViewerUserId: userID,
		Limit:        int32(limit),
		Cursor:       cursor,
	})
	if err != nil {
		h.log.Error("failed to list replies", "error", err, "parent_id", parentID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to fetch replies"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"replies":    res.Replies,
		"nextCursor": res.NextCursor,
		"hasMore":    res.HasMore,
	})
}
