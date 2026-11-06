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

// findIntentsFixtureDir creates a temporary directory containing yaml, yml and
// non-yaml files, plus a nested directory with an additional yaml file.
func findIntentsFixtureDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "nested"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "summarize.yaml"), []byte("name: Summarize"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "feed.yml"), []byte("name: Feed"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("not an intent"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "nested", "explain.yaml"), []byte("name: Explain"), 0644))
	return dir
}

// TestFindIntents tests the recursive .(yaml|yml) file search.
func TestFindIntents(t *testing.T) {
	dir := findIntentsFixtureDir(t)

	intents := findIntents(dir)
	assert.ElementsMatch(t, []string{
		filepath.Join(dir, "summarize.yaml"),
		filepath.Join(dir, "feed.yml"),
		filepath.Join(dir, "nested", "explain.yaml"),
	}, intents)
}

// TestFindIntents_MissingDirectory tests that an empty file list is returned
// for a directory that does not exist.
func TestFindIntents_MissingDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	assert.Empty(t, findIntents(missing))
}

// TestFindIntent tests intent file lookup by intent name.
func TestFindIntent(t *testing.T) {
	dir := findIntentsFixtureDir(t)

	// findIntent relies on the package-level includePaths variable.
	oldIncludePaths := includePaths
	includePaths = []string{dir}
	defer func() { includePaths = oldIncludePaths }()

	t.Run("found", func(t *testing.T) {
		file, err := findIntent("summarize")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(dir, "summarize.yaml"), file)
	})

	t.Run("not_found", func(t *testing.T) {
		_, err := findIntent("unknown")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "Intent Not Found for 'unknown'")
	})
}

// TestLoadIntent tests intent configuration loading from file.
func TestLoadIntent(t *testing.T) {
	t.Run("missing_file", func(t *testing.T) {
		_, err := loadIntent(filepath.Join(t.TempDir(), "missing.yaml"))
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "File not found")
	})

	t.Run("valid_file", func(t *testing.T) {
		cfgFile := filepath.Join(t.TempDir(), "summarize.yaml")
		content := "name: Summarize Intent\nprompt: Summarize {{.Url}}"
		require.NoError(t, os.WriteFile(cfgFile, []byte(content), 0644))

		intent, err := loadIntent(cfgFile)
		require.NoError(t, err)
		require.NotNil(t, intent)
		assert.Equal(t, "Summarize Intent", intent.Name())
	})
}

// TestNewCmd tests command wiring: usage, flags and parse behavior.
func TestNewCmd(t *testing.T) {
	intentdCmd := NewCmd()
	require.NotNil(t, intentdCmd)

	t.Run("usage", func(t *testing.T) {
		assert.Equal(t, "intentd <intent> [input]", intentdCmd.Use)
		assert.Equal(t, "intentd runs agentic Intents", intentdCmd.Short)
	})

	t.Run("flag_defaults", func(t *testing.T) {
		includeFlag := intentdCmd.Flags().Lookup("include")
		require.NotNil(t, includeFlag)
		assert.Equal(t, "I", includeFlag.Shorthand)
		assert.Equal(t, "[./intents/]", includeFlag.DefValue)

		debugFlag := intentdCmd.Flags().Lookup("debug")
		require.NotNil(t, debugFlag)
		assert.Equal(t, "D", debugFlag.Shorthand)
		assert.Equal(t, "false", debugFlag.DefValue)
	})

	t.Run("unknown_flags_allowed", func(t *testing.T) {
		// DATA/FIELDS are passed to intents as unknown flags, e.g.
		// `intentd feed --url https://evi.as/feed/blog.xml`.
		assert.True(t, intentdCmd.FParseErrWhitelist.UnknownFlags)
	})
}
