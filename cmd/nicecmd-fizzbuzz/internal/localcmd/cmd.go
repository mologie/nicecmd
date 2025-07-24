package localcmd

import (
	"fmt"
	"github.com/mologie/nicecmd"
	"github.com/mologie/nicecmd/cmd/nicecmd-fizzbuzz/internal/logutil"
	"github.com/spf13/cobra"
	"io"
	"log/slog"
	"time"
)

type Config struct {
	Limit int `usage:"stop fizzbuzzing at this number"`
}

func Create(parent *cobra.Command) *cobra.Command {
	return nicecmd.SubCommand(parent, nicecmd.Run(run), cobra.Command{
		Use:   "local [--limit <num>] [fizz text] [buzz text]",
		Short: "Fizz and buzz on the local console",
		Args:  cobra.MaximumNArgs(2),
	}, Config{
		Limit: 100,
	})
}

func run(cfg Config, cmd *cobra.Command, args []string) error {
	if cfg.Limit <= 0 {
		return fmt.Errorf("limit must be >0, but got %d", cfg.Limit)
	}

	text := append(args, "Fizz", "Buzz")
	fb := &FizzBuzzer{Fizz: text[0], Buzz: text[1]}

	log := logutil.FromContext(cmd.Context())
	log.Info("local fizzbuzzer starting", slog.Int("limit", cfg.Limit))
	startTime := time.Now()

	fb.Emit(cmd.OutOrStdout(), cfg.Limit)

	log.Info("local fizzbuzzer has completed",
		slog.Duration("duration", time.Since(startTime)))

	return nil
}

type FizzBuzzer struct {
	Fizz string
	Buzz string
}

func (fb *FizzBuzzer) Emit(w io.Writer, limit int) {
	for i := 1; i <= limit; i++ {
		switch {
		case i%15 == 0:
			_, _ = fmt.Fprintln(w, fb.Fizz+fb.Buzz)
		case i%3 == 0:
			_, _ = fmt.Fprintln(w, fb.Fizz)
		case i%5 == 0:
			_, _ = fmt.Fprintln(w, fb.Buzz)
		default:
			_, _ = fmt.Fprintln(w, i)
		}
	}
}
