package handler

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
)

type UserHandler struct {
	userClient userpb.UserServiceClient
	authClient authpb.AuthServiceClient
	log        *slog.Logger
}

func NewUserHandler(
	userClient userpb.UserServiceClient,
	authClient authpb.AuthServiceClient,
	log *slog.Logger,
) *UserHandler {
	return &UserHandler{
		userClient: userClient,
		authClient: authClient,
		log:        log,
	}
}

func (h *UserHandler) Routes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("POST /api/users/{id}/follow", auth(http.HandlerFunc(h.Follow)))
	mux.Handle("DELETE /api/users/{id}/follow", auth(http.HandlerFunc(h.Unfollow)))
	mux.Handle("GET /api/users/suggestions", auth(http.HandlerFunc(h.GetSuggestions)))
	mux.Handle("GET /api/users/{username}", auth(http.HandlerFunc(h.GetProfileByUsername)))
	mux.Handle("GET /api/users/{id}/followers", auth(http.HandlerFunc(h.GetFollowers)))
	mux.Handle("GET /api/users/{id}/following", auth(http.HandlerFunc(h.GetFollowing)))
}

func (h *UserHandler) Follow(w http.ResponseWriter, r *http.Request) {
	followerID := middleware.UserID(r.Context())
	followeeID := r.PathValue("id")
	if followeeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User ID required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.Follow(ctx, &userpb.FollowRequest{
		FollowerId: followerID,
		FolloweeId: followeeID,
	})
	if err != nil {
		h.log.Error("follow failed", "error", err, "follower", followerID, "followee", followeeID)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to follow user"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) Unfollow(w http.ResponseWriter, r *http.Request) {
	followerID := middleware.UserID(r.Context())
	followeeID := r.PathValue("id")
	if followeeID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User ID required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.Unfollow(ctx, &userpb.FollowRequest{
		FollowerId: followerID,
		FolloweeId: followeeID,
	})
	if err != nil {
		h.log.Error("unfollow failed", "error", err, "follower", followerID, "followee", followeeID)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to unfollow user"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) GetProfileByUsername(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	username := r.PathValue("username")
	if username == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Username required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	prof, err := h.authClient.GetProfileByUsername(ctx, &authpb.GetProfileByUsernameRequest{Username: username})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	stats, _ := h.userClient.GetFollowStats(ctx, &userpb.GetFollowStatsRequest{
		TargetUserId: prof.UserId,
		ViewerUserId: viewerID,
	})

	var followerCount, followingCount int32
	var isFollowing bool
	if stats != nil {
		followerCount = stats.FollowerCount
		followingCount = stats.FollowingCount
		isFollowing = stats.IsFollowing
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"userId":            prof.UserId,
		"name":              prof.Name,
		"username":          prof.Username,
		"profilePictureUrl": prof.ProfilePictureUrl,
		"bannerUrl":         prof.BannerUrl,
		"bio":               prof.Bio,
		"createdAt":         prof.CreatedAt,
		"followerCount":     followerCount,
		"followingCount":    followingCount,
		"isFollowing":       isFollowing,
	})
}

func (h *UserHandler) GetFollowers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	targetID := r.PathValue("id")
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

	relRes, err := h.userClient.ListFollowers(ctx, &userpb.ListRelationRequest{
		UserId:       targetID,
		ViewerUserId: viewerID,
		Limit:        int32(limit),
		Cursor:       cursor,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list followers"})
		return
	}

	var userIDs []string
	for _, it := range relRes.Users {
		userIDs = append(userIDs, it.UserId)
	}

	profilesMap := make(map[string]*authpb.GetProfileResponse)
	if len(userIDs) > 0 {
		if pRes, err := h.authClient.GetProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: userIDs}); err == nil {
			for _, p := range pRes.Profiles {
				profilesMap[p.UserId] = p
			}
		}
	}

	var items []map[string]any
	for _, it := range relRes.Users {
		p := profilesMap[it.UserId]
		name := "User"
		uname := "user"
		avatar := ""
		if p != nil {
			name = p.Name
			uname = p.Username
			avatar = p.ProfilePictureUrl
		}
		items = append(items, map[string]any{
			"userId":            it.UserId,
			"name":              name,
			"username":          uname,
			"profilePictureUrl": avatar,
			"isFollowing":       it.IsFollowing,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

func (h *UserHandler) GetFollowing(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	targetID := r.PathValue("id")
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

	relRes, err := h.userClient.ListFollowing(ctx, &userpb.ListRelationRequest{
		UserId:       targetID,
		ViewerUserId: viewerID,
		Limit:        int32(limit),
		Cursor:       cursor,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list following"})
		return
	}

	var userIDs []string
	for _, it := range relRes.Users {
		userIDs = append(userIDs, it.UserId)
	}

	profilesMap := make(map[string]*authpb.GetProfileResponse)
	if len(userIDs) > 0 {
		if pRes, err := h.authClient.GetProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: userIDs}); err == nil {
			for _, p := range pRes.Profiles {
				profilesMap[p.UserId] = p
			}
		}
	}

	var items []map[string]any
	for _, it := range relRes.Users {
		p := profilesMap[it.UserId]
		name := "User"
		uname := "user"
		avatar := ""
		if p != nil {
			name = p.Name
			uname = p.Username
			avatar = p.ProfilePictureUrl
		}
		items = append(items, map[string]any{
			"userId":            it.UserId,
			"name":              name,
			"username":          uname,
			"profilePictureUrl": avatar,
			"isFollowing":       it.IsFollowing,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

func (h *UserHandler) GetSuggestions(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	sugRes, err := h.userClient.SuggestAccounts(ctx, &userpb.SuggestAccountsRequest{
		UserId: viewerID,
		Limit:  3,
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get suggestions"})
		return
	}

	var userIDs []string
	for _, it := range sugRes.Accounts {
		userIDs = append(userIDs, it.UserId)
	}

	profilesMap := make(map[string]*authpb.GetProfileResponse)
	if len(userIDs) > 0 {
		if pRes, err := h.authClient.GetProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: userIDs}); err == nil {
			for _, p := range pRes.Profiles {
				profilesMap[p.UserId] = p
			}
		}
	}

	var items []map[string]any
	for _, it := range sugRes.Accounts {
		p := profilesMap[it.UserId]
		name := "User"
		uname := "user"
		avatar := ""
		if p != nil {
			name = p.Name
			uname = p.Username
			avatar = p.ProfilePictureUrl
		}
		items = append(items, map[string]any{
			"userId":            it.UserId,
			"name":              name,
			"username":          uname,
			"profilePictureUrl": avatar,
			"followerCount":     it.FollowerCount,
			"isFollowing":       false,
		})
	}

	writeJSON(w, http.StatusOK, items)
}
