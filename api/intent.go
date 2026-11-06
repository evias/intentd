package api

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	tmpl "text/template"
	"time"

	"go.yaml.in/yaml/v3"
)

// Intent defines the contract for agentic intents.
type Intent interface {
	// Name should return the intent's name.
	Name() string
	// Template should return the prompt template.
	Template() *tmpl.Template
	// Config should return the intent configuration.
	Config() IntentConfig

	// HasField should return true given an existing field name.
	HasField(field string) bool
	// FieldType should return the type of a field by name,
	// e.g. "binary", "string", "number" or "duration".
	FieldType(name string) string
	// FieldDefault should return the field's default value as a native type.
	FieldDefault(name string) interface{}

	// Prompt should return a prompt with values args.
	Prompt(args any) (string, error)
}

// IntentOption defines the contract for [Driver] option helpers.
type IntentOption func(Intent)

// ----------------------------------------------------------------------------

// agentIntent is a private implementation of the [Intent] interface.
type agentIntent struct {
	name string
	tpl  *tmpl.Template
	conf IntentConfig
}

// Ensure that our implementation satisfies interface.
var _ Intent = (*agentIntent)(nil)

// NewIntent creates an [Intent] instance around conf and options.
func NewIntent(
	conf IntentConfig,
	options ...IntentOption,
) Intent {
	intent := &agentIntent{
		name: conf.Name,
		tpl:  tmpl.Must(tmpl.New("prompt").Parse(conf.Prompt)),
		conf: conf,
	}

	for _, option := range options {
		option(intent)
	}

	return intent
}

// LoadFromConfig loads an [Intent] instance from a YAML configuration file.
// An error is returned if the file does not exist, or if it cannot be read,
// or if the format does not satisfy the unmarshalling to [IntentConfig].
func LoadFromConfig(
	cfgFile string,
	options ...IntentOption,
) (Intent, error) {
	if _, err := os.Stat(cfgFile); err != nil {
		return nil, fmt.Errorf("intent config file not found: %w", err)
	}

	bz, err := os.ReadFile(cfgFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read intent config: %w", err)
	}

	conf := IntentConfig{}
	if err := yaml.Unmarshal(bz, &conf); err != nil {
		return nil, fmt.Errorf("failed to unmarshal intent config: %w", err)
	}

	return NewIntent(conf, options...), nil
}

// Name returns the name as read from [IntentConfig] upon creation.
func (intent *agentIntent) Name() string {
	return intent.name
}

// Template returns the name as read from [IntentConfig] upon creation.
func (intent *agentIntent) Template() *tmpl.Template {
	return intent.tpl
}

// Config returns the injected [IntentConfig].
func (intent *agentIntent) Config() IntentConfig {
	return intent.conf
}

// Prompt returns the parsed template with args, or the empty string with an error.
func (intent *agentIntent) Prompt(args any) (string, error) {
	if err := intent.validateFields(args); err != nil {
		return "", fmt.Errorf("error validating fields: %w", err)
	}

	var (
		argv map[string]string
		err  error
	)
	if argv, err = intent.populateDefaults(args); err != nil {
		return "", fmt.Errorf("error filling default field values: %w", err)
	}

	var sb strings.Builder
	if err = intent.tpl.Execute(&sb, argv); err != nil {
		return "", fmt.Errorf("error building template: %w", err)
	}

	return sb.String(), nil
}

// HasField reads the [IntentConfig] to find a [FieldConfig] by name.
func (intent *agentIntent) HasField(field string) bool {
	_, ok := intent.conf.Fields[field]
	return ok
}

// FieldType returns a string-representation of the type of a field,
// importantly this defines the reflection process for default values.
func (intent *agentIntent) FieldType(name string) string {
	defaultType := "string"
	if !intent.HasField(name) {
		return defaultType
	}

	field := intent.conf.Fields[name]
	return field.Type
}

// FieldDefault returns the field's default value as a native type.
// Supported field types are:
// - `binary`: Returns the empty `byte` slice.
// - `string`: Returns the default field as a `string`.
// - `number`: Returns the default as one of: `float64`, `uint64`, or `int`.
// - `duration`: Returns the default *string* value as a duration, e.g. "3s"=3*time.Second.
// - `attachment`: Returns the empty `string`, i.e. attachments are empty by default.
func (intent *agentIntent) FieldDefault(name string) interface{} {
	if !intent.HasField(name) {
		return []byte{}
	}

	f := intent.conf.Fields[name]
	ft := intent.FieldType(name)

	switch {
	case ft == "binary":
		return []byte{}
	case ft == "string":
		str, ok := f.Default.(string)
		if !ok {
			return ""
		}
		return str
	case ft == "number":
		iF, okF := f.Default.(float64)
		if okF && iF != 0 {
			return iF
		} // else if the value is 0|float64, see end of case block.

		// accept uint64, potentially overturn non-specified int
		// conversion from auto float64=0 ("resolves to 0").
		iUI, okUI := f.Default.(uint64)
		if okUI {
			return iUI
		}
		// fallback to native integer
		iI, okI := f.Default.(int)
		if okI {
			return iI
		}
		// f.Default is 0|float64 or not specified.
		return float64(0)

	case ft == "duration":
		if f.Default == nil {
			return time.Duration(0)
		}

		dur, err := time.ParseDuration(f.Default.(string))
		if err != nil {
			return time.Duration(0)
		}

		return dur

	// Note: attachment field types are empty by default.
	case ft == "attachment":
		return ""
	}

	// Fallback to empty slice of bytes.
	return []byte{}
}

// ----------------------------------------------------------------------------
// Private implementation

// validateFields verifies the presence of all required fields.
//
// TODO(evias): This method expects args with upper-first field names, define as interface.
func (intent *agentIntent) validateFields(args any) error {
	asMap, ok := args.(map[string]string)
	if !ok {
		return fmt.Errorf("invalid args; expected map[string]string, got %T", args)
	}

	for field, cfg := range intent.Config().Fields {
		// FIXME: use a consistent field name interface.
		fieldName := UpperFirstFieldName(field)
		var (
			sv    string
			isSet bool
		)
		if sv, isSet = asMap[fieldName]; cfg.Required && !isSet {
			return fmt.Errorf("invalid args: missing argument '--%s' (%v)", fieldName, asMap)
		}

		if !cfg.AllowEmpty && len(sv) == 0 {
			return fmt.Errorf("invalid args: argument '--%s' may not be empty", field)
		}
	}

	return nil
}

// populateDefaults replaces empty strings with potential default values.
//
// TODO(evias): This method expects args with upper-first field names, define as interface.
// TODO(evias): The emptiness condition only works for the empty string and empty byte slices.
func (intent *agentIntent) populateDefaults(args any) (out map[string]string, err error) {
	argv, ok := args.(map[string]string)
	if !ok {
		err = fmt.Errorf("invalid args; expected map[string]string, got %T", args)
		return // out, err
	}

	out = make(map[string]string, len(argv))
	for field := range intent.Config().Fields {
		// FIXME: use a consistent field name interface.
		fn := UpperFirstFieldName(field)
		ft := intent.FieldType(field)
		sv := argv[fn]

		// FIXME: emptiness condition for duration.
		if len(sv) == 0 {
			fd := intent.FieldDefault(field)
			switch {
			case ft == "binary":
				out[fn] = fmt.Sprintf("%02x", fd)
			case ft == "number":
				if iF, okF := fd.(float64); okF {
					out[fn] = strconv.FormatFloat(iF, 'g', -1, 64)
				} else if iUI, okUI := fd.(uint64); okUI {
					out[fn] = strconv.FormatUint(iUI, 10)
				} else {
					out[fn] = strconv.Itoa(fd.(int))
				}
			default:
				out[fn] = intent.FieldDefault(field).(string)
			}
		} else {
			out[fn] = sv
		}
	}

	return // out, err
}
