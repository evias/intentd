package cmd

import "github.com/spf13/cobra"

var visionCmd = &cobra.Command{
	Use:   "vision [input]",
	Short: "Explain what is to be seen on a picture",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runIntent("vision", args)
	},
}
