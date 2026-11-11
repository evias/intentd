/*
Package intentd implements a terminal user interface to run agentic Intents.

An intent is a pre-configured prompt with arguments that is defined using
a structured YAML-format to permit rapid fields validation and prompt generation.

The `intentd` tool features a suite of pre-defined intents that may be run
repetitively using your preferred cron/PM system.

Usage:

	intentd <intent> [flags]

	Flags:
	-k, --apiKey string         Sets the API Key used for the OpenAI client connection. Required.
	-U, --apiUrl string         Sets the API URL used for the OpenAI client connection (optional). (default "https://ai.chatboxai.app/v1")
	-D, --debug                 Sets whether to enable debug mode/logs or not (optional).
	-h, --help                  help for intentd
	-I, --include stringArray   Specify folders to include for the intents search (optional). (default [intents/])
	-m, --model string          Sets the model that will be used for this conversation (optional). (default "gva/gemini-3.8-flash")
	-o, --output string         Sets an output file to which the response will be appended (optional).
	-Q, --quiet                 Sets whether to enable quiet mode or not (optional).

Examples:

	# fetch a feed of posts and write an overview
	intentd feed --url https://evi.as/feed/blog.xml --label "Grégory's Bits"

	# Or run it *quietly* and *append* output to a file
	intentd feed --url https://evi.as/feed/blog.xml --label "Grégory's Bits" --quiet --output /tmp/gregorys.bits.md

	# explain the content of a website or document (ELI5)
	intentd explain --topic "How computers work"

License:

Copyright 2026 Grégory Saive <greg@evi.as> for re:Software S.L. (resoftware.es).

Licensed under the 3-Clause BSD License.
*/
package main
