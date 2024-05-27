package nicecmd

import (
	"encoding"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"net"
	"os"
	"reflect"
	"slices"
	"strings"
	"time"
)

// Environment is a kill-switch for BindConfig to disable environment variable processing.
// Set this globally if you use another library for environment variables, e.g. Viper.
var Environment = true

const (
	// optPersistent adds the flag to the persistent flag set instead of the command flag set.
	// Persistent flags are a Cobra feature where the parameter is allowed to appear anywhere, not
	// just at the parent command that defined it.
	optPersistent = "persistent"

	// optRequired marks a flag as required
	optRequired = "required"
)

const (
	encodingBase64 = "base64"
	encodingCSV    = "csv"
	encodingCount  = "count"
	encodingHex    = "hex"
	encodingRaw    = "raw"
)

// BindConfig maps fields of cfg to flag sets of cmd. A field's value is set with the following
// precedence: Explicit flag, environment variable, then whatever is already set in cfg.
//
// Struct tags:
// - flag: Set of the flags defined above, separated by commas.
// - param: "foo,f" for --foo=bar or -f x. Defaults to kebab-case of field name without short name.
// - encoding: Type-specific encoding, e.g. "base64" for []byte.
// - env: Environment variable name, "-" for none, defaults to prefixed screaming snake case.
// - usage: Flag usage string. Environment variable name is appended if set.
//
// The env prefix defaults to envPrefix + "_". For structs, the prefix is further extended with the
// screaming snake case of the field name where the struct is embedded.
func BindConfig(envPrefix string, cmd *cobra.Command, cfg any) bool {
	if envPrefix != "" {
		if strings.ToUpper(envPrefix) != envPrefix {
			panic("envPrefix must be all uppercase")
		}
		if strings.HasSuffix(envPrefix, "_") {
			panic("envPrefix must not end with an underscore, it is added automatically")
		}
		envPrefix += "_"
	}
	v := reflect.ValueOf(cfg)
	if v.Kind() != reflect.Ptr || v.Elem().Kind() != reflect.Struct {
		panic("cfg must be a struct pointer")
	}
	var fail bool
	recurseStruct("", envPrefix, fieldOpts{}, cmd, v.Elem(), &fail)
	return !fail
}

func recurseStruct(paramPrefix, envPrefix string, parentOpts fieldOpts,
	cmd *cobra.Command, struct_ reflect.Value, fail *bool,
) {
	type_ := struct_.Type()
	for i := 0; i < type_.NumField(); i++ {
		tags := getFieldTags(paramPrefix, envPrefix, type_.Field(i))
		opts := tags.Opts().Or(parentOpts)
		value := struct_.Field(i)

		var fs *pflag.FlagSet
		if opts.persistent {
			fs = cmd.PersistentFlags()
		} else {
			fs = cmd.Flags()
		}

		// Register with flag set
		// You can add support for custom types by implementing textUmarshalledFlag or pflag.Value.
		// If I happened to miss a type that is supported by spf13/pflag, please let me know and
		// I'll add it here. However, custom or other stdlib types won't be supported directly by
		// matching their type here, as that would require adding additional packages.
		in := value.Addr().Interface()
		switch p := in.(type) {
		case *bool:
			fs.BoolVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]bool:
			fs.BoolSliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]byte:
			switch tags.encoding {
			case encodingBase64:
				fs.BytesBase64VarP(p, tags.name, tags.abbrev, *p, tags.usage)
			case encodingHex:
				fs.BytesHexVarP(p, tags.name, tags.abbrev, *p, tags.usage)
			default:
				panic(fmt.Sprintf(`expected encoding:"base64" or encoding:"hex" for bytes slice %q, got encoding %q`, tags.name, tags.encoding))
			}
		case *int:
			switch tags.encoding {
			case "":
				fs.IntVarP(p, tags.name, tags.abbrev, *p, tags.usage)
			case encodingCount:
				fs.CountVarP(p, tags.name, tags.abbrev, tags.usage)
				if tags.HasEnv() {
					panic(fmt.Sprintf(`count encoding for %q requires env:"-", cannot count env vars`, tags.name))
				}
			default:
				panic(fmt.Sprintf(`expected no encoding or encoding:"count" for int %q, got encoding %q`, tags.name, tags.encoding))
			}
		case *[]int:
			fs.IntSliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *int8:
			fs.Int8VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *int16:
			fs.Int16VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *int32:
			fs.Int32VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]int32:
			fs.Int32SliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *int64:
			fs.Int64VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]int64:
			fs.Int64SliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *uint:
			fs.UintVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]uint:
			fs.UintSliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *uint8:
			fs.Uint8VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *uint16:
			fs.Uint16VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *uint32:
			fs.Uint32VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *uint64:
			fs.Uint64VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *float32:
			fs.Float32VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]float32:
			fs.Float32SliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *float64:
			fs.Float64VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]float64:
			fs.Float64SliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *string:
			fs.StringVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]string:
			// NB: There also is StringArrayVarP, which has nothing to do with arrays, but avoids
			// splitting the string value by commas and appends repeated commands to the slice
			// instead. This is usually desirable, but does not work with environment variables,
			// which can only be set once. Thus default to StringSliceVarP.
			switch tags.encoding {
			case "", encodingCSV:
				fs.StringSliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
			case encodingRaw:
				fs.StringArrayVarP(p, tags.name, tags.abbrev, *p, tags.usage)
				if tags.HasEnv() {
					panic(fmt.Sprintf(`encoding:"raw" for string slice %q requires env:"-"`, tags.name))
				}
			default:
				panic(fmt.Sprintf(`expected encoding:"csv" or encoding:"raw" for string slice %q, got encoding %q`, tags.name, tags.encoding))
			}
		case *map[string]int:
			fs.StringToIntVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *map[string]int64:
			fs.StringToInt64VarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *map[string]string:
			fs.StringToStringVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *time.Duration:
			fs.DurationVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *[]time.Duration:
			fs.DurationSliceVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *net.IP:
			fs.IPVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *net.IPMask:
			fs.IPMaskVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		case *net.IPNet:
			fs.IPNetVarP(p, tags.name, tags.abbrev, *p, tags.usage)
		default:
			if pFlag, ok := in.(pflag.Value); ok {
				// A bunch of libraries, such as K8s, use pflag.Value for various types that also
				// get used as flags with Cobra in frontend tools. This is a catch-all for those.
				fs.VarP(pFlag, tags.name, tags.abbrev, tags.usage)
			} else if textFlag, ok := in.(textUnmarshalledFlag); ok {
				// This is our magic extension point, where any TextUnmarshaler+Stringer can become
				// a flag if it additionally defines CmdTypeDesc() for help messages. The latter
				// method also avoids accidentally flag-i-fying a type that is not meant to be one.
				fs.VarP(newTextValue(textFlag), tags.name, tags.abbrev, tags.usage)
			} else if value.Kind() == reflect.Struct && value.Type().NumField() > 0 {
				recurseStruct(tags.name+"-", tags.env+"_", opts, cmd, value, fail)
				continue // do not process an environment variable
			} else {
				panic(fmt.Sprintf("unsupported field type %T", p))
			}
		}

		param := fs.Lookup(tags.name)
		if param == nil {
			panic(fmt.Sprintf("flag %q not found after it was added", tags.name))
		}

		if opts.required {
			if err := cobra.MarkFlagRequired(fs, param.Name); err != nil {
				panic(fmt.Sprintf("failed to mark flag %q as required: %s", tags.name, err))
			}
			if len(param.Usage) != 0 {
				param.Usage += " "
			}
			param.Usage += "(required)"
		}

		// Apply environment variable
		//goland:noinspection GoBoolExpressions
		if Environment && tags.HasEnv() {
			if len(param.Usage) != 0 {
				param.Usage += " "
			}
			if envVal := os.Getenv(tags.env); envVal != "" {
				ansiColor := "32" // green
				if err := param.Value.Set(envVal); err != nil {
					cmd.Printf("Error: environment variable %s: %s\n", tags.env, err)
					*fail = true
					ansiColor = "31" // red
				}
				param.Changed = true
				param.Usage += fmt.Sprintf("(\033[%smenv %s=%q\033[0m)", ansiColor, tags.env, envVal)
			} else {
				param.Usage += fmt.Sprintf("(env %s)", tags.env)
			}
		}
	}
}

type fieldOpts struct {
	persistent bool
	required   bool
}

func (opts fieldOpts) Or(other fieldOpts) (result fieldOpts) {
	result.persistent = opts.persistent || other.persistent
	result.required = opts.required || other.required
	return
}

type fieldTags struct {
	opts     []string
	encoding string
	name     string
	abbrev   string
	env      string
	usage    string
}

func getFieldTags(paramPrefix, envPrefix string, field reflect.StructField) (tags fieldTags) {
	tags.opts = strings.Split(field.Tag.Get("flag"), ",")
	tags.encoding = field.Tag.Get("encoding")
	tags.name, tags.abbrev, _ = strings.Cut(field.Tag.Get("param"), ",")
	tags.env = field.Tag.Get("env")
	tags.usage = field.Tag.Get("usage")

	if len(tags.name) == 1 {
		if tags.abbrev != "" {
			panic(fmt.Sprintf("param %q must be at least two characters", tags.name))
		}
		tags.abbrev = tags.name
		tags.name = ""
	}
	if tags.name == "" {
		tags.name = paramPrefix + slug(field.Name, '-')
	} else {
		tags.name = paramPrefix + tags.name
	}

	if len(tags.abbrev) > 1 {
		panic(fmt.Sprintf("abbreviation %q for %q must be a single character", tags.abbrev, tags.name))
	}

	if tags.env == "" {
		tags.env = envPrefix + screamingSnake(field.Name)
	} else if tags.env != strings.ToUpper(tags.env) {
		panic(fmt.Sprintf("env tag %q for %q must be uppercase", tags.env, tags.name))
	}

	return
}

func (ft fieldTags) hasOption(name string) bool {
	return slices.Contains(ft.opts, name)
}

func (ft fieldTags) Opts() (opts fieldOpts) {
	opts.persistent = ft.hasOption(optPersistent)
	opts.required = ft.hasOption(optRequired)
	return
}

func (ft fieldTags) HasEnv() bool {
	return ft.env != "-"
}

type textUnmarshalledFlag interface {
	encoding.TextUnmarshaler
	String() string
	CmdTypeDesc() string
}

// textValue implements pflag.Value for textUnmarshalledFlag.
type textValue struct {
	textUnmarshalledFlag
}

func newTextValue(p textUnmarshalledFlag) *textValue {
	return &textValue{textUnmarshalledFlag: p}
}

func (d *textValue) Set(s string) error {
	return d.UnmarshalText([]byte(s))
}

func (d *textValue) Type() string {
	return d.CmdTypeDesc()
}
