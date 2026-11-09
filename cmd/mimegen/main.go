// Command mimegen generates a Go MimeTypeID registry from mime-db JSON.
// Usage:
//
//	mimegen [-in db.json|-] [-url https://...] [-pkg mimeid] [-o mimeids_gen.go]
//
// Examples:
//
//	# from a local file
//	mimegen -in db.json -pkg api -o api/mimeids_gen.go
//
//	# from the network (pin a ref!)
//	mimegen -url https://raw.githubusercontent.com/jshttp/mime-db/v1.54.0/db.json
//
//	# from stdin
//	curl -sL .../db.json | mimegen -
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	in := flag.String("in", "", "path to mime-db db.json, or '-' for stdin")
	url := flag.String("url", "", "URL to fetch db.json from (overrides -in)")
	pkg := flag.String("pkg", "mimeid", "package name for the generated file")
	out := flag.String("o", "mimeids_gen.go", "output file path ('-' for stdout)")
	flag.Parse()

	src := *in
	if *url != "" {
		src = *url
	}
	if src == "" {
		fmt.Fprintln(os.Stderr, "mimegen: provide -in, -url, or pipe JSON via stdin")
		os.Exit(2)
	}

	db, err := Load(src)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mimegen: %v\n", err)
		os.Exit(1)
	}

	code, err := Render(*pkg, db)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mimegen: %v\n", err)
		os.Exit(1)
	}

	if *out == "-" {
		os.Stdout.Write(code)
	} else {
		if err := os.WriteFile(*out, code, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "mimegen: %v\n", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stderr, "mimegen: wrote %s (%d types)\n", *out, len(db))
	}
}
