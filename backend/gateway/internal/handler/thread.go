package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
	"github.com/vaniakarlita/ay-com/backend/gateway/threadpb"
)

type ThreadHandler struct {
	threadClient threadpb.ThreadServiceClient
	authClient   authpb.AuthServiceClient
	log          *slog.Logger
}

func NewThreadHandler(
	threadClient threadpb.ThreadServiceClient,
	authClient authpb.AuthServiceClient,
	log *slog.Logger,
) *ThreadHandler {
	return &ThreadHandler{
		threadClient: threadClient,
		authClient:   authClient,
		log:          log,
	}
}

type createThreadPayload struct {
	Content         string   `json:"content"`
	ReplyPermission string   `json:"replyPermission"`
	Categories      []string `json:"categories"`
	CommunityID     string   `json:"communityId"`
	ScheduledAt     string   `json:"scheduledAt"`
}

func (h *ThreadHandler) Routes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.HandleFunc("GET /api/categories", h.GetCategories)
	mux.Handle("POST /api/threads", auth(http.HandlerFunc(h.CreateThread)))
	mux.Handle("GET /api/threads/feed", auth(http.HandlerFunc(h.ListFeed)))
	mux.Handle("DELETE /api/threads/{id}", auth(http.HandlerFunc(h.DeleteThread)))
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

	// Fetch author details from auth service
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
