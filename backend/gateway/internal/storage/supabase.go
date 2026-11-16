package storage

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type Client struct {
	baseURL string
	key     string
	bucket  string
	http    *http.Client
}

func New(baseURL, key, bucket string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		key:     key,
		bucket:  bucket,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Enabled() bool {
	return c.baseURL != "" && c.key != ""
}

func (c *Client) Upload(ctx context.Context, path, contentType string, data []byte) (string, error) {
	endpoint := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, c.bucket, path)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.key)
	req.Header.Set("apikey", c.key)
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("x-upsert", "false")

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return "", fmt.Errorf("storage returned %d: %s", res.StatusCode, string(body))
	}

	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.baseURL, c.bucket, path), nil
}