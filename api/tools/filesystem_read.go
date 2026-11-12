package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

// This initialization helper feeds custom mime types into stdlib because
// the builtin types are minimal and don't include most modern extensions.
//
// Note that this is *not* the list of supported file types, it is instead
// an extension to the system's (minimal) builtin mime types coverage.
func init() {
	// idempotent; only sets if not already defined
	mime.AddExtensionType(".md", "text/markdown; charset=utf-8")
	mime.AddExtensionType(".markdown", "text/markdown; charset=utf-8")
	mime.AddExtensionType(".json", "application/json")
	mime.AddExtensionType(".yaml", "text/yaml")
	mime.AddExtensionType(".yml", "text/yaml")
	mime.AddExtensionType(".toml", "text/plain; charset=utf-8")
	mime.AddExtensionType(".ts", "text/typescript")
	mime.AddExtensionType(".md", "text/markdown; charset=utf-8")
	mime.AddExtensionType(".markdown", "text/markdown; charset=utf-8")
	mime.AddExtensionType(".csv", "text/csv; charset=utf-8")
	mime.AddExtensionType(".tsv", "text/tab-separated-values; charset=utf-8")
	mime.AddExtensionType(".log", "text/plain; charset=utf-8")
	mime.AddExtensionType(".rst", "text/plain; charset=utf-8")
	mime.AddExtensionType(".rtf", "application/rtf")
	mime.AddExtensionType(".ini", "text/plain; charset=utf-8")
	mime.AddExtensionType(".conf", "text/plain; charset=utf-8")
	mime.AddExtensionType(".cfg", "text/plain; charset=utf-8")
}

// filesystem_read is an [unexported] internal tool to read a file from the fs.
// Note that this tool will only accept content types that represent *text*,
// i.e. "text/*" and "application/json".
// Use [CallInternalTool] to execute this tool.
// The argsJSON parameter must contain valid JSON, with required fields: path.
// This function implements [ToolFunc].
func filesystem_read(
	ctx context.Context,
	argsJSON string,
) (string, error) {
	// Strictly accept only a path argument for this tool.
	var args struct {
		Path string `json:"path"`
	}
	decoder := json.NewDecoder(strings.NewReader(argsJSON))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&args); err != nil {
		return "", fmt.Errorf("arguments failed to decode: %w", err)
	}

	if _, err := os.Stat(args.Path); err != nil {
		return "", fmt.Errorf("File not found: %s", args.Path)
	}

	var (
		contentType   string
		fileExtension = filepath.Ext(args.Path)
	)
	if len(fileExtension) > 0 {
		// Determine content type by file extension, see init().
		if contentType = mime.TypeByExtension(fileExtension); contentType == "" {
			contentType = "application/octet-stream"
		}
	} else {
		// Best bet here is to assume text content,
		// e.g. LICENSE, Makefile, etc.
		contentType = "text/plain; charset=utf-8"
	}

	if !strings.HasPrefix(contentType, "text/") && contentType != "application/json" {
		return "", fmt.Errorf(
			"unsupported content type; expected text/* or application/json, got %s", contentType)
	}

	bz, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}

	return string(bz), nil
}
