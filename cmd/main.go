package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use: "intentd <intent> [input]",
	Short: "intentd runs agentic Intents",
	Long: `intentd is a terminal user interface to run agentic Intents.

An intent is a specific activity that an (L|S|V)LM agent executes on a
repetitive basis, or once, and which is controlled and guarded by human
review processes.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(summarizeCmd)
	rootCmd.AddCommand(feedCmd)
	rootCmd.AddCommand(visionCmd)
	rootCmd.AddCommand(explainCmd)
}

func runIntent(name string, args []string) {
	input := ""
	if len(args) > 0 {
		input = args[0]
	}
	fmt.Printf("intent: %s, input: %s\n", name, input)
}
