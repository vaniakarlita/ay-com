package handler

import (
	"context"
	"net/http"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
	"github.com/vaniakarlita/ay-com/backend/gateway/internal/middleware"
)

type loginBody struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type tokenBody struct {
	RefreshToken string `json:"refreshToken"`
}

func (h *AuthHandler) SessionRoutes(mux *http.ServeMux, auth func(http.Handler) http.Handler) {
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("POST /api/auth/refresh", h.Refresh)
	mux.HandleFunc("POST /api/auth/logout", h.Logout)
	mux.Handle("GET /api/me", auth(http.HandlerFunc(h.Me)))
	mux.Handle("PUT /api/users/me", auth(http.HandlerFunc(h.UpdateProfile)))
	mux.Handle("POST /api/users/me/deactivate", auth(http.HandlerFunc(h.DeactivateAccount)))
}

func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var body loginBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.Login(ctx, &authpb.LoginRequest{Email: body.Email, Password: body.Password})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"accessToken":  res.AccessToken,
		"refreshToken": res.RefreshToken,
	})
}

func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var body tokenBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.RefreshToken(ctx, &authpb.RefreshTokenRequest{RefreshToken: body.RefreshToken})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"accessToken":  res.AccessToken,
		"refreshToken": res.RefreshToken,
	})
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	var body tokenBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.Logout(ctx, &authpb.LogoutRequest{RefreshToken: body.RefreshToken})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": res.Message})
}

func (h *AuthHandler) Me(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.GetProfile(ctx, &authpb.GetProfileRequest{UserId: middleware.UserID(r.Context())})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"userId":            res.UserId,
		"name":              res.Name,
		"username":          res.Username,
		"profilePictureUrl": res.ProfilePictureUrl,
		"bannerUrl":         res.BannerUrl,
		"bio":               res.Bio,
		"isPrivate":         res.IsPrivate,
		"createdAt":         res.CreatedAt,
	})
}

type updateProfileBody struct {
	Name              *string `json:"name"`
	Username          *string `json:"username"`
	Email             *string `json:"email"`
	Gender            *string `json:"gender"`
	DateOfBirth       *string `json:"dateOfBirth"`
	Bio               *string `json:"bio"`
	ProfilePictureURL *string `json:"profilePictureUrl"`
	BannerURL         *string `json:"bannerUrl"`
	IsPrivate         *bool   `json:"isPrivate"`
	OldPassword       *string `json:"oldPassword"`
	NewPassword       *string `json:"newPassword"`
}

func (h *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	var body updateProfileBody
	if !decode(w, r, &body) {
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	req := &authpb.UpdateProfileRequest{
		UserId:            userID,
		Name:              body.Name,
		Username:          body.Username,
		Email:             body.Email,
		Gender:            body.Gender,
		DateOfBirth:       body.DateOfBirth,
		Bio:               body.Bio,
		ProfilePictureUrl: body.ProfilePictureURL,
		BannerUrl:         body.BannerURL,
		IsPrivate:         body.IsPrivate,
		OldPassword:       body.OldPassword,
		NewPassword:       body.NewPassword,
	}

	res, err := h.client.UpdateProfile(ctx, req)
	if err != nil {
		h.fail(w, err)
		return
	}

	respMap := map[string]any{
		"user": map[string]any{
			"userId":            res.Profile.UserId,
			"name":              res.Profile.Name,
			"username":          res.Profile.Username,
			"profilePictureUrl": res.Profile.ProfilePictureUrl,
			"bannerUrl":         res.Profile.BannerUrl,
			"bio":               res.Profile.Bio,
			"isPrivate":         res.Profile.IsPrivate,
			"createdAt":         res.Profile.CreatedAt,
		},
	}
	if res.NewAccessToken != nil && res.NewRefreshToken != nil {
		respMap["accessToken"] = *res.NewAccessToken
		respMap["refreshToken"] = *res.NewRefreshToken
	}

	writeJSON(w, http.StatusOK, respMap)
}

type deactivateBody struct {
	Password string `json:"password"`
}

func (h *AuthHandler) DeactivateAccount(w http.ResponseWriter, r *http.Request) {
	var body deactivateBody
	if !decode(w, r, &body) {
		return
	}

	userID := middleware.UserID(r.Context())
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.DeactivateAccount(ctx, &authpb.DeactivateAccountRequest{
		UserId:   userID,
		Password: body.Password,
	})
	if err != nil {
		h.fail(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"success": res.Success,
		"message": res.Message,
	})
}
