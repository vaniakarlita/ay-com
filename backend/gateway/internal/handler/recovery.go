package handler

import (
	"context"
	"net/http"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
)

type answerBody struct {
	Email  string `json:"email"`
	Answer string `json:"answer"`
}

type resetBody struct {
	Email       string `json:"email"`
	ResetToken  string `json:"resetToken"`
	NewPassword string `json:"newPassword"`
}

func (h *AuthHandler) RecoveryRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/forgot/question", h.ForgotQuestion)
	mux.HandleFunc("POST /api/auth/forgot/verify", h.ForgotVerify)
	mux.HandleFunc("POST /api/auth/forgot/reset", h.ForgotReset)
}

func (h *AuthHandler) ForgotQuestion(w http.ResponseWriter, r *http.Request) {
	var body emailBody
	if !decode(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.GetSecurityQuestion(ctx, &authpb.GetSecurityQuestionRequest{Email: body.Email})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"question": res.Question})
}

func (h *AuthHandler) ForgotVerify(w http.ResponseWriter, r *http.Request) {
	var body answerBody
	if !decode(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.VerifySecurityAnswer(ctx, &authpb.VerifySecurityAnswerRequest{Email: body.Email, Answer: body.Answer})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"resetToken": res.ResetToken})
}

func (h *AuthHandler) ForgotReset(w http.ResponseWriter, r *http.Request) {
	var body resetBody
	if !decode(w, r, &body) {
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.ResetPassword(ctx, &authpb.ResetPasswordRequest{
		Email:       body.Email,
		ResetToken:  body.ResetToken,
		NewPassword: body.NewPassword,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": res.Message})
}