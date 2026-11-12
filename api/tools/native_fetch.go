package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	MaxFetchPayload = 1 << 24 // 16 Mb sanity cap for now
)

// native_fetch is an [unexported] internal tool to fetch a URL using HTTP.
// A MaxFetchPayload limit of bytes is set to read the body. This should be
// way above what is usually necessary and covers images/audio as well.
// Use [CallInternalTool] to execute this tool.
// The argsJSON parameter must contain valid JSON, with required fields: url.
// This function implements [ToolFunc].
func native_fetch(
	ctx context.Context,
	argsJSON string,
) (string, error) {
	// Strictly accept only an url argument for this tool.
	var args struct {
		Url string `json:"url"`
	}
	decoder := json.NewDecoder(strings.NewReader(argsJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return "", fmt.Errorf("arguments failed to decode: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, args.Url, nil)
	if err != nil {
		return "", err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	// pre-check size
	if resp.ContentLength > MaxFetchPayload {
		return "", fmt.Errorf("content-length %d exceeds max", resp.ContentLength)
	}

	// sanity cap of 16Mb stops reading further
	limitReader := io.NopCloser(io.LimitReader(resp.Body, MaxFetchPayload+1))
	body, err := io.ReadAll(limitReader)
	if err != nil {
		return "", fmt.Errorf("response body read error: %w", err)
	}
	if len(body) > MaxFetchPayload {
		return "", fmt.Errorf("response exceeds %d bytes", MaxFetchPayload)
	}
	return string(body), nil
}
