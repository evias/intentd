package cmd

import "github.com/spf13/cobra"

var feedCmd = &cobra.Command{
	Use:   "feed [input]",
	Short: "Fetch latest entries from a feed and build insights",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runIntent("feed", args)
	},
}
