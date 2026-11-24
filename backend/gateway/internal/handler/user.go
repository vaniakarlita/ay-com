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
	mux.Handle("POST /api/users/{id}/block", auth(http.HandlerFunc(h.BlockUser)))
	mux.Handle("DELETE /api/users/{id}/block", auth(http.HandlerFunc(h.UnblockUser)))
	mux.Handle("POST /api/users/{id}/report", auth(http.HandlerFunc(h.ReportUser)))
	mux.Handle("GET /api/users/me/blocked", auth(http.HandlerFunc(h.ListBlockedUsers)))
	mux.Handle("GET /api/users/me/notifications", auth(http.HandlerFunc(h.GetNotificationSettings)))
	mux.Handle("PUT /api/users/me/notifications", auth(http.HandlerFunc(h.UpdateNotificationSettings)))
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

func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
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

	relRes, err := h.userClient.ListBlockedUsers(ctx, &userpb.ListRelationRequest{
		UserId: viewerID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("list blocked users failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list blocked users"})
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
			"createdAt":         it.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

type notifSettingsBody struct {
	NotifLike      bool `json:"notifLike"`
	NotifRepost    bool `json:"notifRepost"`
	NotifFollow    bool `json:"notifFollow"`
	NotifMention   bool `json:"notifMention"`
	NotifCommunity bool `json:"notifCommunity"`
}

func (h *UserHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.GetNotificationSettings(ctx, &userpb.GetNotificationSettingsRequest{UserId: viewerID})
	if err != nil {
		h.log.Error("get notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	var body notifSettingsBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UpdateNotificationSettings(ctx, &userpb.UpdateNotificationSettingsRequest{
		UserId:         viewerID,
		NotifLike:      body.NotifLike,
		NotifRepost:    body.NotifRepost,
		NotifFollow:    body.NotifFollow,
		NotifMention:   body.NotifMention,
		NotifCommunity: body.NotifCommunity,
	})
	if err != nil {
		h.log.Error("update notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
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

func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
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

	relRes, err := h.userClient.ListBlockedUsers(ctx, &userpb.ListRelationRequest{
		UserId: viewerID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("list blocked users failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list blocked users"})
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
			"createdAt":         it.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

type notifSettingsBody struct {
	NotifLike      bool `json:"notifLike"`
	NotifRepost    bool `json:"notifRepost"`
	NotifFollow    bool `json:"notifFollow"`
	NotifMention   bool `json:"notifMention"`
	NotifCommunity bool `json:"notifCommunity"`
}

func (h *UserHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.GetNotificationSettings(ctx, &userpb.GetNotificationSettingsRequest{UserId: viewerID})
	if err != nil {
		h.log.Error("get notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	var body notifSettingsBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UpdateNotificationSettings(ctx, &userpb.UpdateNotificationSettingsRequest{
		UserId:         viewerID,
		NotifLike:      body.NotifLike,
		NotifRepost:    body.NotifRepost,
		NotifFollow:    body.NotifFollow,
		NotifMention:   body.NotifMention,
		NotifCommunity: body.NotifCommunity,
	})
	if err != nil {
		h.log.Error("update notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
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

	var isBlocking, isBlockedBy bool
	if viewerID != "" && viewerID != prof.UserId {
		blkRes, _ := h.userClient.CheckBlockStatus(ctx, &userpb.CheckBlockRequest{
			UserA: viewerID,
			UserB: prof.UserId,
		})
		if blkRes != nil {
			isBlocking = blkRes.IsBlocking
			isBlockedBy = blkRes.IsBlockedBy
		}
	}

	if isBlockedBy {
		writeJSON(w, http.StatusOK, map[string]any{
			"userId":            prof.UserId,
			"name":              prof.Name,
			"username":          prof.Username,
			"profilePictureUrl": "",
			"bannerUrl":         "",
			"bio":               "",
			"createdAt":         prof.CreatedAt,
			"followerCount":     0,
			"followingCount":    0,
			"isFollowing":       false,
			"isBlocking":        isBlocking,
			"isBlockedBy":       true,
		})
		return
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
		"isBlocking":        isBlocking,
		"isBlockedBy":       false,
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

	if targetID != viewerID {
		tProf, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: targetID})
		if err == nil && tProf != nil && tProf.IsPrivate {
			stats, sErr := h.userClient.GetFollowStats(ctx, &userpb.GetFollowStatsRequest{
				TargetUserId: targetID,
				ViewerUserId: viewerID,
			})
			if sErr != nil || !stats.IsFollowing {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "This account is private. Follow this account to see their followers."})
				return
			}
		}
	}

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
		if pRes, err := h.authClient.GetActiveProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: userIDs}); err == nil {
			for _, p := range pRes.Profiles {
				profilesMap[p.UserId] = p
			}
		}
	}

	var items []map[string]any
	for _, it := range relRes.Users {
		p := profilesMap[it.UserId]
		if p == nil {
			continue
		}
		items = append(items, map[string]any{
			"userId":            it.UserId,
			"name":              p.Name,
			"username":          p.Username,
			"profilePictureUrl": p.ProfilePictureUrl,
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

	if targetID != viewerID {
		tProf, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: targetID})
		if err == nil && tProf != nil && tProf.IsPrivate {
			stats, sErr := h.userClient.GetFollowStats(ctx, &userpb.GetFollowStatsRequest{
				TargetUserId: targetID,
				ViewerUserId: viewerID,
			})
			if sErr != nil || !stats.IsFollowing {
				writeJSON(w, http.StatusForbidden, map[string]string{"error": "This account is private. Follow this account to see who they follow."})
				return
			}
		}
	}

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
		if pRes, err := h.authClient.GetActiveProfilesByIDs(ctx, &authpb.GetProfilesByIDsRequest{UserIds: userIDs}); err == nil {
			for _, p := range pRes.Profiles {
				profilesMap[p.UserId] = p
			}
		}
	}

	var items []map[string]any
	for _, it := range relRes.Users {
		p := profilesMap[it.UserId]
		if p == nil {
			continue
		}
		items = append(items, map[string]any{
			"userId":            it.UserId,
			"name":              p.Name,
			"username":          p.Username,
			"profilePictureUrl": p.ProfilePictureUrl,
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

type ReportRequestPayload struct {
	Reason  string `json:"reason"`
	Details string `json:"details"`
}

func (h *UserHandler) BlockUser(w http.ResponseWriter, r *http.Request) {
	blockerID := middleware.UserID(r.Context())
	blockedID := r.PathValue("id")
	if blockedID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User ID required"})
		return
	}
	if blockerID == blockedID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot block yourself"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: blockedID})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	res, err := h.userClient.BlockUser(ctx, &userpb.BlockRequest{
		BlockerId: blockerID,
		BlockedId: blockedID,
	})
	if err != nil {
		h.log.Error("block user failed", "error", err, "blocker", blockerID, "blocked", blockedID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to block user"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
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

	relRes, err := h.userClient.ListBlockedUsers(ctx, &userpb.ListRelationRequest{
		UserId: viewerID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("list blocked users failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list blocked users"})
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
			"createdAt":         it.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

type notifSettingsBody struct {
	NotifLike      bool `json:"notifLike"`
	NotifRepost    bool `json:"notifRepost"`
	NotifFollow    bool `json:"notifFollow"`
	NotifMention   bool `json:"notifMention"`
	NotifCommunity bool `json:"notifCommunity"`
}

func (h *UserHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.GetNotificationSettings(ctx, &userpb.GetNotificationSettingsRequest{UserId: viewerID})
	if err != nil {
		h.log.Error("get notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	var body notifSettingsBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UpdateNotificationSettings(ctx, &userpb.UpdateNotificationSettingsRequest{
		UserId:         viewerID,
		NotifLike:      body.NotifLike,
		NotifRepost:    body.NotifRepost,
		NotifFollow:    body.NotifFollow,
		NotifMention:   body.NotifMention,
		NotifCommunity: body.NotifCommunity,
	})
	if err != nil {
		h.log.Error("update notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UnblockUser(w http.ResponseWriter, r *http.Request) {
	blockerID := middleware.UserID(r.Context())
	blockedID := r.PathValue("id")
	if blockedID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User ID required"})
		return
	}
	if blockerID == blockedID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot unblock yourself"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UnblockUser(ctx, &userpb.BlockRequest{
		BlockerId: blockerID,
		BlockedId: blockedID,
	})
	if err != nil {
		h.log.Error("unblock user failed", "error", err, "blocker", blockerID, "blocked", blockedID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to unblock user"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
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

	relRes, err := h.userClient.ListBlockedUsers(ctx, &userpb.ListRelationRequest{
		UserId: viewerID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("list blocked users failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list blocked users"})
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
			"createdAt":         it.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

type notifSettingsBody struct {
	NotifLike      bool `json:"notifLike"`
	NotifRepost    bool `json:"notifRepost"`
	NotifFollow    bool `json:"notifFollow"`
	NotifMention   bool `json:"notifMention"`
	NotifCommunity bool `json:"notifCommunity"`
}

func (h *UserHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.GetNotificationSettings(ctx, &userpb.GetNotificationSettingsRequest{UserId: viewerID})
	if err != nil {
		h.log.Error("get notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	var body notifSettingsBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UpdateNotificationSettings(ctx, &userpb.UpdateNotificationSettingsRequest{
		UserId:         viewerID,
		NotifLike:      body.NotifLike,
		NotifRepost:    body.NotifRepost,
		NotifFollow:    body.NotifFollow,
		NotifMention:   body.NotifMention,
		NotifCommunity: body.NotifCommunity,
	})
	if err != nil {
		h.log.Error("update notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) ReportUser(w http.ResponseWriter, r *http.Request) {
	reporterID := middleware.UserID(r.Context())
	reportedID := r.PathValue("id")
	if reportedID == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "User ID required"})
		return
	}
	if reporterID == reportedID {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Cannot report yourself"})
		return
	}

	var req ReportRequestPayload
	if !decode(w, r, &req) {
		return
	}

	if req.Reason == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Report reason is required"})
		return
	}
	if req.Reason == "Other" && req.Details == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Please provide details for reason Other"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	_, err := h.authClient.GetProfile(ctx, &authpb.GetProfileRequest{UserId: reportedID})
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "User not found"})
		return
	}

	res, err := h.userClient.ReportUser(ctx, &userpb.ReportRequest{
		ReporterId: reporterID,
		ReportedId: reportedID,
		Reason:     req.Reason,
		Details:    req.Details,
	})
	if err != nil {
		h.log.Error("report user failed", "error", err, "reporter", reporterID, "reported", reportedID)
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "Failed to submit report or report already pending"})
		return
	}

	writeJSON(w, http.StatusOK, res)
}

func (h *UserHandler) ListBlockedUsers(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
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

	relRes, err := h.userClient.ListBlockedUsers(ctx, &userpb.ListRelationRequest{
		UserId: viewerID,
		Limit:  int32(limit),
		Cursor: cursor,
	})
	if err != nil {
		h.log.Error("list blocked users failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to list blocked users"})
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
			"createdAt":         it.CreatedAt,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"users":      items,
		"nextCursor": relRes.NextCursor,
		"hasMore":    relRes.HasMore,
	})
}

type notifSettingsBody struct {
	NotifLike      bool `json:"notifLike"`
	NotifRepost    bool `json:"notifRepost"`
	NotifFollow    bool `json:"notifFollow"`
	NotifMention   bool `json:"notifMention"`
	NotifCommunity bool `json:"notifCommunity"`
}

func (h *UserHandler) GetNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.GetNotificationSettings(ctx, &userpb.GetNotificationSettingsRequest{UserId: viewerID})
	if err != nil {
		h.log.Error("get notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to get notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}

func (h *UserHandler) UpdateNotificationSettings(w http.ResponseWriter, r *http.Request) {
	viewerID := middleware.UserID(r.Context())
	var body notifSettingsBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.userClient.UpdateNotificationSettings(ctx, &userpb.UpdateNotificationSettingsRequest{
		UserId:         viewerID,
		NotifLike:      body.NotifLike,
		NotifRepost:    body.NotifRepost,
		NotifFollow:    body.NotifFollow,
		NotifMention:   body.NotifMention,
		NotifCommunity: body.NotifCommunity,
	})
	if err != nil {
		h.log.Error("update notification settings failed", "error", err, "user_id", viewerID)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "Failed to update notification settings"})
		return
	}

	writeJSON(w, http.StatusOK, map[string]bool{
		"notifLike":      res.NotifLike,
		"notifRepost":    res.NotifRepost,
		"notifFollow":    res.NotifFollow,
		"notifMention":   res.NotifMention,
		"notifCommunity": res.NotifCommunity,
	})
}
