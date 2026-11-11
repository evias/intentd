# intentd: Agentic cronjobs

This repository implements a terminal user interface to run agentic **Intents**.

An **intent** is a pre-configured *prompt with arguments* that is defined using
a structured YAML-format to permit rapid fields validation and prompt generation.

The `intentd` tool features a suite of pre-defined *intents* that may be run
repetitively using your preferred cron/PM system.

## Usage

```bash
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
```

Use `intentd` to run agentic intents with your provider configuration. 
You may use `intentd` to run any of the pre-configured intents, or
your own intents using the `-I path/to/your/intents` option.

```bash
# fetch a feed of posts and write an overview
intentd feed --url https://evi.as/feed/blog.xml --label "Grégory's Bits"

# explain the content of a website or document (ELI5)
intentd explain --topic "How computers work"
```

## Developer notes

```bash
go build -o intentd main.go    # build the binary
go build -o mimegen cmd/mimegen # build the mimegen tool
./intentd <intent> [input] # run an intent locally
go vet ./...                   # static analysis, must pass before pushing
go test ./... --count=1        # run the unit tests suite
```

### Generating the `MimeTypeID` registry

Use `go generate` to generate up-to-date mime types mappings. It automatically
downloads the `db.json` from: `https://raw.githubusercontent.com/jshttp/mime-db/release-1-54-0/db.json`.

Note that re-generating the `MimeTypeID` registry from branch `master`
of the repository `jshttp/mime-db` may introduce mime types that are not yet
listed in the repository. Pull requests sending *only* new type mappings will
be automatically discarded, we'll use timely Actions for this.

```bash
$ go generate ./...
```

Or use the `mimegen` tool directly. Note that the following command is included
as a `//go generate` build flag already:

```bash
$ go run ./cmd/mimegen -url https://raw.githubusercontent.com/jshttp/mime-db/release-1-54-0/db.json -pkg api -o api/mimeids_gen.go
```

## Examples

```bash
# summarize a single link or document
$ intentd summarize https://evi.as/hire-me

# fetch latest entries from a feed and build insights
$ intentd feed --url https://evi.as/feed/blog.xml

# explain what is to be seen on a picture
$ intentd vision https://evi.as/assets/evias_with_marley_and_maya.png

# ELI5 on any topic
$ intentd explain https://evias.be/assets/bitcoin.pdf
```

## License

Copyright 2026 Grégory Saive <greg@evi.as> for re:Software S.L. (resoftware.es).

Licensed under the [3-Clause BSD License](./LICENSE).

