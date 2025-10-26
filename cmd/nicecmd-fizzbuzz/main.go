// nicecmd-fizzbuzz is an enterprise-grade, highly configurable fizz buzzer
// (everything in this tool is nonsense, but should demonstrate nicecmd usage)
package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/mologie/nicecmd"
	"github.com/mologie/nicecmd/cmd/nicecmd-fizzbuzz/internal/localcmd"
	"github.com/mologie/nicecmd/cmd/nicecmd-fizzbuzz/internal/logutil"
	"github.com/spf13/cobra"
)

type MainConfig struct {
	Log logutil.Config `flag:"persistent"`
}

func main() {
	cmd := nicecmd.RootCommand(nicecmd.Setup(setup), cobra.Command{
		Use:   "fizzbuzz [--log-level <level>] [--log-type <JSON|TEXT>]",
		Short: "Enterprise-grade fizzbuzz (nicecmd demo)",
	}, MainConfig{
		Log: logutil.Config{
			Level:  logutil.Level(slog.LevelInfo),
			Format: logutil.FormatText,
		},
	})

	localcmd.Create(cmd)

	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func setup(cfg *MainConfig, cmd *cobra.Command, args []string) error {
	// This method demonstrates inheriting a log context to child commands.
	// An application could also use slog.SetDefault(), but I'd rather have an
	// invalid default handler and ensure that logging contexts are propagated
	// properly to improve testability of subcommands.
	handler, err := cfg.Log.NewHandler()
	if err != nil {
		return fmt.Errorf("failed to create log handler: %w", err)
	}
	cmd.SetContext(logutil.WithLogContext(cmd.Context(), slog.New(handler)))
	return nil
}
