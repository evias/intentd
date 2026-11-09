// Package intentd implements a terminal user interface to run agentic Intents.
//
// An **intent** is not just a *tool*, it is a specific activity that an (L|S|V)LM
// agent executes on a repetitive basis, or once, and which is controlled and
// guarded by human review processes.
//
// The `intentd` tool features a suite of pre-defined *intents* that may be run
// repetitively using your preferred cron/PM system.
//
// Usage:
//
//	intentd <intent> [input]
//
// Examples:
//
//	# fetch a feed of posts and write an overview
//	intentd feed --url https://evi.as/feed/blog.xml --label "Grégory's Bits"
//
//	# summarize a website or document
//	intentd summarize --url https://evi.as/hire-me
//
//	# explain the content of a website or document (ELI5)
//	intentd explain --url https://evias.be/assets/bitcoin.pdf
//
//	# describe what is to be seen on a picture
//	intentd vision --url https://evias.be/assets/evias_with_marley_and_maya.png
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
