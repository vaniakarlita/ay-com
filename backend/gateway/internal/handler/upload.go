package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log/slog"
	"net/http"
)

const maxUploadBytes = 2 << 20

var allowedImages = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/gif":  ".gif",
	"image/webp": ".webp",
}

type Storage interface {
	Enabled() bool
	Upload(ctx context.Context, path, contentType string, data []byte) (string, error)
}

type UploadHandler struct {
	store Storage
	log   *slog.Logger
}

func NewUploadHandler(store Storage, log *slog.Logger) *UploadHandler {
	return &UploadHandler{store: store, log: log}
}

func (h *UploadHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/uploads/image", h.Image)
}

func (h *UploadHandler) Image(w http.ResponseWriter, r *http.Request) {
	if !h.store.Enabled() {
		writeError(w, http.StatusServiceUnavailable, "image storage is not configured")
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadBytes+(64<<10))
	if err := r.ParseMultipartForm(maxUploadBytes); err != nil {
		writeError(w, http.StatusBadRequest, "image must be 2 MB or smaller")
		return
	}

	kind := r.FormValue("kind")
	if kind != "profile" && kind != "banner" {
		writeError(w, http.StatusBadRequest, "kind must be profile or banner")
		return
	}

	file, _, err := r.FormFile("file")
	if err != nil {
		writeError(w, http.StatusBadRequest, "image file is required")
		return
	}
	defer file.Close()

	data, err := io.ReadAll(io.LimitReader(file, maxUploadBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, "could not read the image")
		return
	}
	if len(data) == 0 || len(data) > maxUploadBytes {
		writeError(w, http.StatusBadRequest, "image must be 2 MB or smaller")
		return
	}

	contentType := http.DetectContentType(data)
	ext, ok := allowedImages[contentType]
	if !ok {
		writeError(w, http.StatusBadRequest, "image must be JPEG, PNG, GIF, or WebP")
		return
	}

	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		h.log.Error("generate file name failed", "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}
	path := kind + "/" + hex.EncodeToString(raw) + ext

	url, err := h.store.Upload(r.Context(), path, contentType, data)
	if err != nil {
		h.log.Error("upload to storage failed", "error", err)
		writeError(w, http.StatusBadGateway, "could not store the image, please try again")
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{"url": url})
}