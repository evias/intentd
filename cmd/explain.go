package cmd

import "github.com/spf13/cobra"

var explainCmd = &cobra.Command{
	Use:   "explain [input]",
	Short: "ELI5 on any topic",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runIntent("explain", args)
	},
}
