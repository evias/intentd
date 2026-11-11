package tools

import (
	"fmt"
	"io"
	"net/http"
)

const (
	MaxFetchPayload = 1 << 24 // 16 Mb sanity cap for now
)

func native_fetch(url string) (string, error) {
	resp, err := http.Get(url)
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
