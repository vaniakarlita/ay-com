package handler

import (
	"context"
	"net/http"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
)

type googleBody struct {
	IDToken string `json:"idToken"`
}

func (h *AuthHandler) GoogleRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/google", h.Google)
}

func (h *AuthHandler) Google(w http.ResponseWriter, r *http.Request) {
	var body googleBody
	if !decode(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.GoogleAuth(ctx, &authpb.GoogleAuthRequest{IdToken: body.IDToken})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"registered":   res.Registered,
		"accessToken":  res.AccessToken,
		"refreshToken": res.RefreshToken,
		"email":        res.Email,
		"name":         res.Name,
	})
}