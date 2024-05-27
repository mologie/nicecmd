package nicecmd

import (
	"github.com/spf13/cobra"
	"os"
)

var osExitOrTestHook = os.Exit

type RunE[T any] func(cfg T, cmd *cobra.Command, args []string) error

type RunFuncs[T any] struct {
	PersistentPreRun  RunE[T]
	PreRun            RunE[T]
	Run               RunE[T]
	PostRun           RunE[T]
	PersistentPostRun RunE[T]
}

func init() {
	// Opinionated default: We'd want all parent hooks to run by default. The user can still
	// disable this after the fact by changing Cobra's global setting back to false.
	cobra.EnableTraverseRunHooks = true
}

// PersistentPreRun is a convenience function to create a RunFuncs with only the PersistentPreRun function set.
func PersistentPreRun[T any](f func(cfg T, cmd *cobra.Command, args []string) error) RunFuncs[T] {
	return RunFuncs[T]{PersistentPreRun: f}
}

// Run is a convenience function to create a RunFuncs with only the Run function set.
func Run[T any](f func(cfg T, cmd *cobra.Command, args []string) error) RunFuncs[T] {
	return RunFuncs[T]{Run: f}
}

func Command[T any](envPrefix string, run RunFuncs[T], cmd cobra.Command, cfg T) *cobra.Command {
	cmd.PersistentPreRunE = passCfg(&cfg, run.PersistentPreRun)
	cmd.PreRunE = passCfg(&cfg, run.PreRun)
	cmd.RunE = passCfg(&cfg, run.Run)
	cmd.PostRunE = passCfg(&cfg, run.PostRun)
	cmd.PersistentPostRunE = passCfg(&cfg, run.PersistentPostRun)

	// Opinionated defaults: Local flags should just work, and the user is expected to provide a
	// proper "Use" line for the command that suggests where flags should go.
	if cmd.Use == "" {
		panic("use line must be set, and should include all non-global flags")
	}
	cmd.TraverseChildren = true
	cmd.DisableAutoGenTag = true
	cmd.DisableFlagsInUseLine = true

	// Opinionated default: Accept no args unless those were explicitly allowed by the user.
	// pflag's default is to accept arbitrary args by default.
	if cmd.Args == nil {
		cmd.Args = cobra.NoArgs
	}

	if BindConfig(envPrefix, &cmd, &cfg) {
		return &cmd
	} else {
		_ = cmd.Usage()
		osExitOrTestHook(1)
		return nil
	}
}

func passCfg[T any](cfg *T, f RunE[T]) func(cmd *cobra.Command, args []string) error {
	if f != nil {
		return func(cmd *cobra.Command, args []string) error {
			return f(*cfg, cmd, args)
		}
	} else {
		return nil
	}
}
