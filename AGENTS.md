# AGENTS.md

## Project Overview

`intentd` is a terminal user interface to run agentic **Intents**.

An **intent** is not just a *tool*, it is a specific activity that an
(L|S|V)LM agent executes on a repetitive basis, or once, and which is
controlled and guarded by human review processes.

The `intentd` tool features a suite of pre-defined *intents* that may be run
repetitively using your preferred cron/PM system.

- Module name: `evias/intentd`
- Language: Go (1.27+)
- Dependencies: [Cobra](https://github.com/spf13/cobra), [Testify](github.com/stretchr/testify), [YAMLv3](go.yaml.in/yaml/v3), [OpenAI](https://github.com/openai/openai-go)

## Repository Layout

```
.
├── main.go          # package main: binary entrypoint, calls cmd.Execute()
├── cmd/             # package cmd: Cobra command definitions
│   ├── main.go      # rootCmd definition and Execute()
│   ├── summarize.go # "summarize" intent subcommand
│   ├── feed.go      # "feed" intent subcommand
│   ├── vision.go    # "vision" intent subcommand
│   └── explain.go   # "explain" intent subcommand
├── go.mod / go.sum
└── README.md
```

## Development Guidelines

### Building & Running

```bash
go build -o intentd main.go    # build the binary
go build -o mimegen cmd/mimegen # build the mimegen tool
./intentd <intent> [input] # run an intent locally
go vet ./...                   # static analysis, must pass before pushing
go test ./... --count=1        # run the unit tests suite
```

### Code Conventions

- The root command (`rootCmd` in `cmd/main.go`) uses the usage format
  `intentd <intent> [input]`; keep new intent subcommands consistent with it.
- Each intent subcommand lives in its own file under `cmd/` and is registered
  with `rootCmd` via `AddCommand` in `cmd/main.go:init()`.
- Keep external dependencies minimal.
- Do not re-generate the `api/mimeids_gen.go` file, i.e. don't use `go generate`.

### Git Strategy

- Work on *feature branches*, one branch per feature, e.g. `feat/add_summarize_cmd`.
- Push feature branches to the remote and open a PR; PRs are reviewed by the Professor.
- Never commit directly to `main`.

### License

Copyright 2026 Grégory Saive <greg@evi.as> for re:Software S.L. (resoftware.es).
Licensed under the 3-Clause BSD License (see `LICENSE`).
