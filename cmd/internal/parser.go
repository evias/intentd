package internal

import (
	"strings"

	"github.com/evias/intentd/api"
)

// -----------------------------------------------------------------------------
// Parser — parses custom data arguments passed to agentic intents.
// -----------------------------------------------------------------------------

// ParseIntentArgs parses data arguments, typically passed through os.Args in a
// format similar to e.g. `--count 10`, `--limit=10`, or `--debug`.
//
// Returns a map[string]string with keys from [api.IntentConfig#Fields].
// Returns a map with keys upper-first, e.g. "Count", "Limit", and "Debug".
func ParseIntentArgs(
	intent api.Intent,
	args []string,
) (data map[string]string) {
	// v contains "--sleep", "--direction" keys.
	v := sliceToMap(args)

	// data contains "sleep", "direction" keys.
	data = make(map[string]string, len(args))

	// Make sure we have all fields (required).
	for field := range intent.Config().Fields {
		// Note that we *do not accept short form* intent arguments.
		// i.e. Intent arguments are always long form "--url".
		f := "--" + strings.ToLower(field)
		if a, ok := v[f]; ok {
			// Golang text/template expects uppercase-first keys.
			key := api.UpperFirstFieldName(field)
			data[key] = a
		}
	}

	return /* data */
}

// sliceToMap converts a slice of bash-style arguments, e.g. "--speed", into a
// key-value map with string keys and values.
func sliceToMap(args []string) map[string]string {
	m := make(map[string]string)
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			// detects --, but nothing to do
			continue
		}
		if strings.HasPrefix(args[i], "-") {
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				m[args[i]] = args[i+1]
				i++ // consume the value
			} else if strings.Contains(args[i], "=") {
				nv := strings.SplitN(args[i], "=", 2)
				m[nv[0]] = nv[1]
			} else {
				m[args[i]] = "" // flag with no value (or boolean-style flag)
			}
		}
	}
	return m
}
