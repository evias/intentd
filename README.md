# intentd: Agentic cronjobs

This repository implements a terminal user interface to run agentic **Intents**.

An **intent** is not just a *tool*, it is a specific activity that an (L|S|V)LM
agent executes on a repetitive basis, or once, and which is controlled and
guarded by human review processes.

The `intentd` tool features a suite of pre-defined *intents* that may be run
repetitively using your preferred cron/PM system.

## Usage

```bash
Usage: intentd <intent> [input]
```

Use `intentd` to run agentic intents with your provider configuration. Use the
configuration command `intentd config` before your first run, i.e.:

```bash
intentd config
```

After that, you may use `intentd` to run any of the pre-configured intents, or
your own intents using the `-I path/to/your/intents` option.

```bash
# fetch a feed of posts and write an overview
intentd feed --url https://evi.as/feed/blog.xml --label "Grégory's Bits"

# summarize a website or document
intentd summarize --url https://evi.as/hire-me

# explain the content of a website or document (ELI5)
intentd explain --url https://evias.be/assets/bitcoin.pdf

# describe what is to be seen on a picture
intentd vision --url https://evias.be/assets/evias_with_marley_and_maya.png
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

