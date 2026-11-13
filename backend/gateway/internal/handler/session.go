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
	writeJSON(w, http.StatusOK, map[string]string{"userId": middleware.UserID(r.Context())})
}