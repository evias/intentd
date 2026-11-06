package main

import (
	"os"

	"github.com/evias/intentd/cmd"
)

func main() {
	intentd := cmd.NewCmd()
	if err := intentd.Execute(); err != nil {
		os.Exit(1)
	}
}
