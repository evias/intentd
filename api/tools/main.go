package tools

type ToolFunc func(string) (string, error)

var KnownTools = []string{
	"native_fetch",
}

var KnownToolsFn = map[string]ToolFunc{
	"native_fetch": native_fetch,
}
