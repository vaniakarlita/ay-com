package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
	"github.com/vaniakarlita/ay-com/backend/gateway/userpb"
)

func (h *UserHandler) SettingsRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.Handle("GET /api/users/me/blocked", auth(http.HandlerFunc(h.ListBlockedUsers)))
	mux.Handle("GET /api/users/me/notifications", auth(http.HandlerFunc(h.GetNotificationSettings)))
	mux.Handle("PUT /api/users/me/notifications", auth(http.HandlerFunc(h.UpdateNotificationSettings)))
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
