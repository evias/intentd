package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
)

const (
	MaxFetchPayload = 1 << 24 // 16 Mb sanity cap for now
)

func native_fetch(ctx context.Context, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	// pre-check size
	if resp.ContentLength > MaxFetchPayload {
		return "", fmt.Errorf("fetch: content-length %d exceeds max", resp.ContentLength)
	}

	// sanity cap of 16Mb stops reading further
	limitReader := io.NopCloser(io.LimitReader(resp.Body, MaxFetchPayload+1))
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("read body: %w", err)
	}
	if len(body) > MaxFetchPayload {
		return "", fmt.Errorf("fetch: response exceeds %d bytes", MaxFetchPayload)
	}
	return string(body), nil
}
