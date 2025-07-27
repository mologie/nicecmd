package nicecmd

import (
	"bufio"
	"encoding"
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"net"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"
)

type AllTypesConfig struct {
	Bool           bool              `expect:"--bool * (env TEST_BOOL)" usage:"*"`
	Bools          []bool            `expect:"--bools bools * (env TEST_BOOLS) (default [])" usage:"*"`
	BytesHex       []byte            `expect:"--bytes-base64 bytesBase64 * (env TEST_BYTES_BASE64)" usage:"*" encoding:"hex"`
	BytesBase64    []byte            `expect:"--bytes-hex bytesHex * (env TEST_BYTES_HEX)" usage:"*" encoding:"base64"`
	Int            int               `expect:"-i, --integer int * (env TESTINTEGER)" usage:"*" param:"integer,i" env:"TESTINTEGER"`
	IntCount       int               `expect:"--int-count count *" usage:"*" encoding:"count" env:"-"`
	Ints           []int             `expect:"--ints ints * (env TEST_INTS)" usage:"*"` // NB: inconsistent usage, special treatment happens in pflag/flag.go
	Int8           int8              `expect:"--int8 int8 * (env TEST_INT8)" usage:"*"`
	Int16          int16             `expect:"--int16 int16 * (env TEST_INT16)" usage:"*"`
	Int32          int32             `expect:"--int32 int32 * (env TEST_INT32)" usage:"*"`
	Ints32         []int32           `expect:"--ints32 int32Slice * (env TEST_INTS32) (default [])" usage:"*"`
	Int64          int64             `expect:"--int64 int * (env TEST_INT64)" usage:"*"`
	Ints64         []int64           `expect:"--ints64 int64Slice * (env TEST_INTS64) (default [])" usage:"*"`
	Uint           uint              `expect:"--uint uint * (env TEST_UINT)" usage:"*"`
	Uints          []uint            `expect:"--uints uints * (env TEST_UINTS) (default [])" usage:"*"`
	Uint8          uint8             `expect:"--uint8 uint8 * (env TEST_UINT8)" usage:"*"`
	Uint16         uint16            `expect:"--uint16 uint16 * (env TEST_UINT16)" usage:"*"`
	Uint32         uint32            `expect:"--uint32 uint32 * (env TEST_UINT32)" usage:"*"`
	Uint64         uint64            `expect:"--uint64 uint * (env TEST_UINT64)" usage:"*"`
	Float32        float32           `expect:"--float32 float32 * (env TEST_FLOAT32)" usage:"*"`
	Floats32       []float32         `expect:"--floats32 float32Slice * (env TEST_FLOATS32) (default [])" usage:"*"`
	Float64        float64           `expect:"--float64 float * (env TEST_FLOAT64)" usage:"*"`
	Floats64       []float64         `expect:"--floats64 float64Slice * (env TEST_FLOATS64) (default [])" usage:"*"`
	String         string            `expect:"--string string * (env TEST_STRING)" usage:"*"`
	StringsCSV     []string          `expect:"--strings-csv strings * (env TEST_STRINGS_CSV)" usage:"*"`
	StringsRaw     []string          `expect:"--strings-raw stringArray *" usage:"*" encoding:"raw" env:"-"`
	StringToInt    map[string]int    `expect:"--string-to-int stringToInt * (env TEST_STRING_TO_INT) (default [])" usage:"*"`
	StringToInt64  map[string]int64  `expect:"--string-to-int64 stringToInt64 * (env TEST_STRING_TO_INT64) (default [])" usage:"*"`
	StringToString map[string]string `expect:"--string-to-string stringToString * (env TEST_STRING_TO_STRING) (default [])" usage:"*"`
	Duration       time.Duration     `expect:"--duration duration * (env TEST_DURATION)" usage:"*"`
	Durations      []time.Duration   `expect:"--durations durationSlice * (env TEST_DURATIONS) (default [])" usage:"*"`
	IP             net.IP            `expect:"--ip ip * (env TEST_IP)" usage:"*"`
	IPMask         net.IPMask        `expect:"--ip-mask ipMask * (env TEST_IP_MASK)" usage:"*"`
	IPNet          net.IPNet         `expect:"--ip-net ipNet * (env TEST_IP_NET)" usage:"*"`
	PFlagValue     pflagValue        `expect:"--pflag-value pflagValue * (env TEST_PFLAG_VALUE)" param:"pflag-value" env:"TEST_PFLAG_VALUE" usage:"*"`
	TextValue      textValue         `expect:"--text-value * (env TEST_TEXT_VALUE)" param:"text-value" env:"TEST_TEXT_VALUE" usage:"*"`
}

var _ pflag.Value = &pflagValue{}

type pflagValue struct{ val string }

func (p *pflagValue) Set(s string) error { p.val = s; return nil }
func (p *pflagValue) String() string     { return p.val }
func (p *pflagValue) Type() string       { return "pflagValue" }

var (
	_ encoding.TextUnmarshaler = &textValue{}
	_ encoding.TextMarshaler   = &textValue{}
)

type textValue struct{ val string }

func (p *textValue) UnmarshalText(text []byte) error { p.val = string(text); return nil }
func (p *textValue) MarshalText() ([]byte, error)    { return []byte(p.val), nil }

func TestBindConfig_AllTypes(t *testing.T) {
	// This test is pretty cheesy, (ab)using the fact that the FlagUsages() method accesses most of
	// the stuff relevant to nicecmd. I would not call it elegant, but it's compact.
	// Caveats with this approach are that there is no 1:1 mapping between fields and expected help,
	// and that changes in Cobra that substantially modify help output will break this test.

	var cfg AllTypesConfig
	cmd := &cobra.Command{}
	BindConfig(cmd, &cfg, WithEnvPrefix("TEST"))
	cmd.Flags().VisitAll(func(flag *pflag.Flag) {
		if annotations, ok := flag.Annotations[annotationEnv]; ok {
			flag.Usage += fmt.Sprintf(" (env %s)", annotations[0])
		}
	})

	// Extract "expect" tags via reflection on cfg
	expected := make(map[string]struct{})
	confType := reflect.ValueOf(cfg).Type()
	for i := 0; i < confType.NumField(); i++ {
		field := confType.Field(i)
		if expect, ok := field.Tag.Lookup("expect"); ok {
			expected[expect] = struct{}{}
		} else {
			t.Errorf("field %s has no expect tag", field.Name)
		}
	}
	if len(expected) != confType.NumField() {
		t.Error("there is at least one duplicate expect tag")
		return
	}

	// Now verify that normalized generated usage lines exactly match the expect tags
	usage := cmd.Flags().FlagUsages()
	scanner := bufio.NewScanner(strings.NewReader(usage))
	spaces := regexp.MustCompile(` +`)
	for scanner.Scan() {
		line := strings.TrimLeft(scanner.Text(), " ")
		line = spaces.ReplaceAllString(line, " ")
		if _, ok := expected[line]; ok {
			delete(expected, line)
		} else {
			t.Errorf("unexpected flag in usage: %s", line)
		}
	}
	if len(expected) > 0 {
		for k := range expected {
			t.Errorf("flag missing from usage: %s", k)
		}
	}
}

func TestBindConfig_Nested(t *testing.T) {
	type Embed struct {
		Embed int
	}
	var conf struct {
		Embed
		Level1 struct {
			Outer  bool `usage:"*"`
			Level2 struct {
				Inner pflagValue `usage:"*"`
			} `flag:"persistent"`
		} `flag:"required"`
	}
	cmd := &cobra.Command{}
	BindConfig(cmd, &conf, WithEnvPrefix("TEST"))
	fs := cmd.Flags()
	pfs := cmd.PersistentFlags()

	if err := fs.Set("level1-outer", "true"); err != nil {
		t.Errorf("set outer: %v", err)
	}
	if _, ok := fs.Lookup("level1-outer").Annotations[cobra.BashCompOneRequiredFlag]; !ok {
		t.Error("outer should be required")
	}
	if !conf.Level1.Outer {
		t.Error("outer bool should be true")
	}

	if err := pfs.Set("level1-level2-inner", "foo"); err != nil {
		t.Errorf("set inner: %v", err)
	}
	if conf.Level1.Level2.Inner.val != "foo" {
		t.Errorf(`inner value mismatch, expected "foo", got %q`, conf.Level1.Level2.Inner.val)
	}

	if flag := fs.Lookup("embed"); flag != nil {
		if env := flag.Annotations[annotationEnv]; len(env) == 0 || env[0] != "TEST_EMBED" {
			t.Errorf("expected env var for embed to be TEST_EMBED, got %q", env)
		}
	} else {
		t.Error("expected embed flag to be present")
	}

	if flag := pfs.Lookup("level1-level2-inner"); flag != nil {
		if env := flag.Annotations[annotationEnv]; len(env) == 0 || env[0] != "TEST_LEVEL1_LEVEL2_INNER" {
			t.Errorf("expected env var for inner to be TEST_LEVEL1_LEVEL2_INNER, got %q", env)
		}
	} else {
		t.Error("expected level1-level2-inner flag to be present")
	}
}

func TestBindConfig_ExplicitEnv(t *testing.T) {
	type conf struct {
		String string `env:"TEST_STRING" usage:"*"`
		Int    int    `usage:"*"`
	}
	cmd := &cobra.Command{}
	BindConfig(cmd, &conf{})
	fs := cmd.Flags()
	if env := fs.Lookup("string").Annotations[annotationEnv]; len(env) == 0 || env[0] != "TEST_STRING" {
		t.Errorf("expected env var for string to be TEST_STRING, got %q", env)
	}
	if _, ok := fs.Lookup("int").Annotations[annotationEnv]; ok {
		t.Error("expected no env var for int, but found one")
	}
}

func TestBindConfig_InvalidEnvPrefix(t *testing.T) {
	benignCmd := &cobra.Command{}
	expectPanic(t, "must not end with an underscore", func() {
		BindConfig(benignCmd, &struct{}{}, WithEnvPrefix("TEXT_"))
	})
	expectPanic(t, "must be all uppercase", func() {
		BindConfig(benignCmd, &struct{}{}, WithEnvPrefix("TeST"))
	})
}

func TestBindConfig_InvalidConfigTags(t *testing.T) {
	type unsupported string
	tt := []struct {
		name  string
		panic string
		conf  any
	}{
		{name: "bad non-pointer input", panic: "must be a struct pointer", conf: struct{}{}},
		{name: "bad bytes encoding", panic: `got encoding "count"`, conf: &struct {
			Bytes []byte `encoding:"count"`
		}{}},
		{name: "bad int encoding", panic: `got encoding "base64"`, conf: &struct {
			Int int `encoding:"base64"`
		}{}},
		{name: "bad string slice encoding", panic: `got encoding "base64"`, conf: &struct {
			String []string `encoding:"base64"`
		}{}},
		{name: "raw string slice with env", panic: `requires env:"-"`, conf: &struct {
			String []string `encoding:"raw"`
		}{}},
		{name: "counted int with env", panic: `requires env:"-"`, conf: &struct {
			Int int `encoding:"count"`
		}{}},
		{name: "bad type", panic: "unsupported field type *nicecmd.unsupported", conf: &struct {
			Unsupported unsupported
		}{}},
		{name: "bad env name", panic: "must be in SCREAMING_SNAKE_CASE", conf: &struct {
			String string `env:"lowercase"`
		}{}},
		{name: "bad abbreviation", panic: "must be a single character", conf: &struct {
			String string `param:"foo,bar"`
		}{}},
		{name: "bad param", panic: "must be at least two characters", conf: &struct {
			String string `param:"f,b"`
		}{}},
	}
	for _, test := range tt {
		t.Run(test.name, func(t *testing.T) {
			expectPanic(t, test.panic, func() {
				BindConfig(&cobra.Command{},
					test.conf,
					WithEnvPrefix("TEST"))
			})
		})
	}
}

func expectPanic(t *testing.T, message string, f func()) {
	t.Helper()
	defer func() {
		t.Helper()
		if r := recover(); r == nil {
			t.Error("expected panic")
		} else if !strings.Contains(r.(string), message) {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	f()
}
