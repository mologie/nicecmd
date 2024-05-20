package logutil

import (
	"log/slog"
)

type Config struct {
	Level  Level  `flag:"optional" usage:"TRACE, DEBUG, INFO, WARN, or ERROR"`
	Format Format `usage:"TEXT or JSON"`
}

func (c Config) NewHandler() (slog.Handler, error) {
	return NewHandler(c.Format, c.Level)
}
