package cmd

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEnsureIncludePaths tests validation of --include folder paths.
func TestEnsureIncludePaths(t *testing.T) {
	t.Run("empty_paths", func(t *testing.T) {
		assert.NoError(t, ensureIncludePaths([]string{}))
	})
	t.Run("existing_directory", func(t *testing.T) {
		assert.NoError(t, ensureIncludePaths([]string{t.TempDir()}))
	})
	t.Run("missing_directory", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "missing")
		err := ensureIncludePaths([]string{missing})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--include")
	})
	t.Run("not_a_directory", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "file.txt")
		require.NoError(t, os.WriteFile(file, []byte("content"), 0644))
		err := ensureIncludePaths([]string{file})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}
