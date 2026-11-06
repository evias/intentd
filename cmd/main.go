package cmd

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/evias/intentd/api"
	"github.com/evias/intentd/cmd/internal"
)

const (
	defaultIncludePath = "intents/"
)

var (
	includePaths = []string{}
	intentName   string
	enableDebug  bool
)

func NewCmd() *cobra.Command {
	intentdCmd := &cobra.Command{
		Use:   "intentd <intent> [input]",
		Short: "intentd runs agentic Intents",
		Long: `intentd is a terminal user interface to run agentic Intents.

An intent is a specific activity that an (L|S|V)LM agent executes on a
repetitive basis, or once, and which is controlled and guarded by human
review processes.`,
		Args: cobra.MinimumNArgs(1),
		// This PreRunE ensures that --include options are used with actual
		// folder paths, to avoid any work to happen before this validation.
		// It also updates the default logger if necessary (--debug).
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := ensureIncludePaths(includePaths); err != nil {
				return err
			}

			// --debug enables debug messages in default logger.
			logLevel := new(slog.LevelVar)
			if enableDebug {
				logLevel.Set(slog.LevelDebug)
				handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
					Level: logLevel,
				})
				slog.SetDefault(slog.New(handler))
			}

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			intentName = args[0]

			// If user does not explicitely change the include paths, add intents/.
			if len(includePaths) == 0 {
				includePaths = []string{defaultIncludePath}
			}

			var (
				intentFile string
				intent     api.Intent
			)

			// Lookup the intent using includePaths
			if intentFile, err = findIntent(intentName); err != nil {
				return err
			}

			// Load the intent configuration file (YAML) into a [api.Intent].
			if intent, err = loadIntent(intentFile); err != nil {
				return err
			}

			// e.g. everything after "feed" in: `intentd feed --count=3`
			dataArgs := os.Args[2:]
			promptArgs := internal.ParseIntentArgs(intent, dataArgs)
			delete(promptArgs, "debug")
			delete(promptArgs, "include")

			slog.Debug("Intent loaded",
				"intent", intentName,
				"args", fmt.Sprintf("%v", promptArgs))

			var (
				prompt  string
				argsErr error
			)
			if prompt, argsErr = intent.Prompt(promptArgs); argsErr != nil {
				return argsErr
			}

			slog.Debug("Prompt template executed", "prompt", prompt)
			return nil
		},
	}

	// Allow unknown flags to accept DATA/FIELDS being passed to intent.
	// e.g. `intentd feed --url https://evi.as/feed/blog.xml`
	intentdCmd.FParseErrWhitelist.UnknownFlags = true

	intentdCmd.Flags().StringArrayVarP(&includePaths, "include", "I", []string{"./intents/"},
		"Specify folders to include for the intents search (optional).")
	intentdCmd.Flags().BoolVarP(&enableDebug, "debug", "D", false,
		"Sets whether to enable debug mode/logs or not (optional).")

	return intentdCmd
}

// ----------------------------------------------------------------------------
// Private implementation

// findIntent searches for intentName using includePaths.
func findIntent(intentName string) (string, error) {
	for _, includePath := range includePaths {
		intents := findIntents(includePath)
		for _, f := range intents {
			file := filepath.Base(f)
			if strings.TrimSuffix(file, filepath.Ext(file)) == intentName {
				return f, nil
			}
		}
	}

	return "", fmt.Errorf("Intent Not Found for '%s'", intentName)
}

// loadIntent returns a [api.Intent] after reading intantName from includePaths.
func loadIntent(intentFile string) (api.Intent, error) {
	// The configuration file must exist and be accessible
	if _, err := os.Stat(intentFile); err != nil {
		return nil, fmt.Errorf("File not found: %s", intentFile)
	}

	return api.LoadFromConfig(intentFile)
}

// ensureIncludePaths ensures that --include options are valid by returning an
// error if an entry is not a directory, or if it doesn't exist.
func ensureIncludePaths(paths []string) error {
	for _, dir := range paths {
		info, err := os.Stat(dir)
		if err != nil {
			return fmt.Errorf("--include %q: %w", dir, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("--include %q: not a directory", dir)
		}
	}
	return nil
}

// findIntents walks includePath to search for .(yaml|yml) files.
func findIntents(includePath string) []string {
	files := []string{}
	filepath.WalkDir(includePath, func(
		path string,
		d fs.DirEntry,
		err error,
	) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		searchName := strings.ToLower(d.Name())
		if strings.HasSuffix(searchName, ".yaml") || strings.HasSuffix(searchName, ".yml") {
			files = append(files, path)
		}

		return nil
	})

	return files
}
