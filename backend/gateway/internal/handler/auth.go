package handler

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/vaniakarlita/ay-com/backend/gateway/authpb"
)

const (
	maxBodyBytes   = 1 << 20
	requestTimeout = 5 * time.Second
)

type AuthHandler struct {
	client authpb.AuthServiceClient
	log    *slog.Logger
}

func NewAuthHandler(client authpb.AuthServiceClient, log *slog.Logger) *AuthHandler {
	return &AuthHandler{client: client, log: log}
}

func (h *AuthHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/verify", h.Verify)
	mux.HandleFunc("POST /api/auth/resend-code", h.ResendCode)
	mux.HandleFunc("GET /api/health", h.Health)
}

type registerBody struct {
	Name              string `json:"name"`
	Username          string `json:"username"`
	Email             string `json:"email"`
	Password          string `json:"password"`
	Gender            string `json:"gender"`
	DateOfBirth       string `json:"dateOfBirth"`
	ProfilePictureURL string `json:"profilePictureUrl"`
	BannerURL         string `json:"bannerUrl"`
	SecurityQuestion  string `json:"securityQuestion"`
	SecurityAnswer    string `json:"securityAnswer"`
	Newsletter        bool   `json:"newsletter"`
}

type verifyBody struct {
	Email string `json:"email"`
	Code  string `json:"code"`
}

type emailBody struct {
	Email string `json:"email"`
}

func (h *AuthHandler) Health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var body registerBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.Register(ctx, &authpb.RegisterRequest{
		Name:              body.Name,
		Username:          body.Username,
		Email:             body.Email,
		Password:          body.Password,
		Gender:            body.Gender,
		DateOfBirth:       body.DateOfBirth,
		ProfilePictureUrl: body.ProfilePictureURL,
		BannerUrl:         body.BannerURL,
		SecurityQuestion:  body.SecurityQuestion,
		SecurityAnswer:    body.SecurityAnswer,
		Newsletter:        body.Newsletter,
	})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"userId": res.UserId, "message": res.Message})
}

func (h *AuthHandler) Verify(w http.ResponseWriter, r *http.Request) {
	var body verifyBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.VerifyEmail(ctx, &authpb.VerifyEmailRequest{Email: body.Email, Code: body.Code})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": res.Message})
}

func (h *AuthHandler) ResendCode(w http.ResponseWriter, r *http.Request) {
	var body emailBody
	if !decode(w, r, &body) {
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
	defer cancel()

	res, err := h.client.ResendCode(ctx, &authpb.ResendCodeRequest{Email: body.Email})
	if err != nil {
		h.fail(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": res.Message})
}

func (h *AuthHandler) fail(w http.ResponseWriter, err error) {
	st, ok := status.FromError(err)
	if !ok {
		h.log.Error("unexpected error", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	switch st.Code() {
	case codes.InvalidArgument:
		writeError(w, http.StatusBadRequest, st.Message())
	case codes.AlreadyExists:
		writeError(w, http.StatusConflict, st.Message())
	case codes.NotFound:
		writeError(w, http.StatusNotFound, st.Message())
	case codes.FailedPrecondition:
		writeError(w, http.StatusUnprocessableEntity, st.Message())
	case codes.PermissionDenied:
		writeError(w, http.StatusForbidden, st.Message())
	case codes.ResourceExhausted:
		writeError(w, http.StatusTooManyRequests, st.Message())
	case codes.Unauthenticated:
		writeError(w, http.StatusUnauthorized, st.Message())
	case codes.DeadlineExceeded, codes.Unavailable:
		h.log.Error("auth service unavailable", "code", st.Code().String())
		writeError(w, http.StatusServiceUnavailable, "service temporarily unavailable")
	default:
		h.log.Error("auth service error", "code", st.Code().String(), "message", st.Message())
		writeError(w, http.StatusInternalServerError, "internal server error")
	}
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return false
	}
	return true
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, code int, message string) {
	writeJSON(w, code, map[string]string{"error": message})
}