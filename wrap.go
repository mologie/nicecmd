package nicecmd

import (
	"github.com/spf13/cobra"
	"os"
	"strings"
)

var osExitOrTestHook = os.Exit

// Hook matches cobra.Command's various RunE functions, plus a config to be bound via Command.
type Hook[T any] func(cfg T, cmd *cobra.Command, args []string) error

// Hooks provides various RunE functions of cobra.Command. Cobra's naming scheme is reused here.
// For clarity, only the "persistent" hooks run even when a subcommand is called.
type Hooks[T any] struct {
	PersistentPreRun  Hook[T]
	PreRun            Hook[T]
	Run               Hook[T]
	PostRun           Hook[T]
	PersistentPostRun Hook[T]
}

func init() {
	// Opinionated default: We'd want all parent hooks to run by default. The user can still
	// disable this after the fact by changing Cobra's global setting back to false.
	cobra.EnableTraverseRunHooks = true
}

// Setup is a convenience function to create Hooks with only the PersistentPreRun function set.
// This function is executed for sub-commands too, but before argument validation.
func Setup[T any](f Hook[T]) Hooks[T] {
	return Hooks[T]{PersistentPreRun: f}
}

// Run is a convenience function to create Hooks with only the Run function set.
// This function is executed only when the user runs this specific command.
func Run[T any](f Hook[T]) Hooks[T] {
	return Hooks[T]{Run: f}
}

// SetupAndRun is a convenience function that combines Setup and Run.
func SetupAndRun[T any](setup Hook[T], run Hook[T]) Hooks[T] {
	return Hooks[T]{
		PersistentPreRun: setup,
		Run:              run,
	}
}

func RootCommand[T any](hooks Hooks[T], cmd cobra.Command, cfg T, opts ...Option) *cobra.Command {
	return SubCommand(nil, hooks, cmd, cfg, opts...)
}

func SubCommand[T any](parent *cobra.Command, hooks Hooks[T], cmd cobra.Command, cfg T, opts ...Option) *cobra.Command {
	cmd.PersistentPreRunE = passCfg(&cfg, hooks.PersistentPreRun)
	cmd.PreRunE = passCfg(&cfg, hooks.PreRun)
	cmd.RunE = passCfg(&cfg, hooks.Run)
	cmd.PostRunE = passCfg(&cfg, hooks.PostRun)
	cmd.PersistentPostRunE = passCfg(&cfg, hooks.PersistentPostRun)

	// Local flags should just work, and the user is expected to provide a proper "Use" line for
	// the command that suggests where flags should go.
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

	// An option to register a parent is offered so that we know the full env path before binding
	// to the configuration structure.
	if parent != nil {
		parent.AddCommand(&cmd)
	}

	// Default to the command's path as env prefix. Can be overridden by an additional option.
	var names []string
	cmd.VisitParents(func(parent *cobra.Command) {
		names = append(names, parent.Name())
	})
	defaultName := screamingSnake(strings.Join(append(names, cmd.Name()), "_"))
	opts = append([]Option{WithEnvPrefix(defaultName)}, opts...)

	if BindConfig(&cmd, &cfg, opts...) {
		return &cmd
	} else {
		_ = cmd.Usage()
		osExitOrTestHook(1)
		return nil
	}
}

func passCfg[T any](cfg *T, f Hook[T]) func(cmd *cobra.Command, args []string) error {
	if f != nil {
		return func(cmd *cobra.Command, args []string) error {
			return f(*cfg, cmd, args)
		}
	} else {
		return nil
	}
}
