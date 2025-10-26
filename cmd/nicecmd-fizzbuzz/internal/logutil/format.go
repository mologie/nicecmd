package logutil

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/spf13/pflag"
)

var _ pflag.Value = func() *Format { return nil }()

type Format string

func (f *Format) Set(s string) error {
	format := Format(strings.ToUpper(s))
	switch format {
	case FormatText, FormatJSON:
		*f = format
		return nil
	}
	return fmt.Errorf("invalid log format text: %q", s)
}

func (f *Format) String() string {
	return string(*f)
}

func (f *Format) Type() string {
	return "format"
}

const (
	FormatText Format = "TEXT"
	FormatJSON Format = "JSON"
)

func NewHandler(format Format, level Level) (slog.Handler, error) {
	out := os.Stderr
	opt := &slog.HandlerOptions{
		Level:       slog.Level(level),
		ReplaceAttr: LevelAttrReplacer,
	}
	switch format {
	case FormatText:
		return slog.NewTextHandler(out, opt), nil
	case FormatJSON:
		return slog.NewJSONHandler(out, opt), nil
	}
	return nil, fmt.Errorf("unsupported log format: %s", format)
}
