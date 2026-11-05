package cmd

import "github.com/spf13/cobra"

var summarizeCmd = &cobra.Command{
	Use:   "summarize [input]",
	Short: "Summarize a single link or document",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runIntent("summarize", args)
	},
}
