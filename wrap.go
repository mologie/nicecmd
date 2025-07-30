package nicecmd

import (
	"github.com/spf13/cobra"
	"strings"
)

// Environment determines whether environment variables are bound and processed.
// Change this globally if you use another library for environment variables, e.g. Viper.
var Environment = true

// Hook matches cobra.Command's various RunE functions, plus a config to be bound via Command.
type Hook[T any] func(cfg *T, cmd *cobra.Command, args []string) error

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
	// We'd want all parent hooks to run by default.
	// This is required for environment variable processing and dotenv support.
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
	return Hooks[T]{PersistentPreRun: setup, Run: run}
}

// RootCommand adds nicecmd-specific persistent arguments to the given command, e.g. --env-file for
// loading dotenv files. It is otherwise identical to Command.
func RootCommand[T any](hooks Hooks[T], cmd cobra.Command, cfg T, opts ...Option) *cobra.Command {
	if Environment {
		hooks.PersistentPreRun = applyEnvToCmd(&cmd, hooks.PersistentPreRun)
		hooks.PersistentPreRun = checkEnv(&cmd, hooks.PersistentPreRun)
		hooks.PersistentPreRun = applyDotEnv(&cmd, hooks.PersistentPreRun)
		cmd.SetHelpFunc(applyEnvToHelp(cmd.HelpFunc()))
		cmd.SetUsageFunc(applyEnvToUsage(cmd.UsageFunc()))
		cmd.PersistentFlags().Bool("env-lax", false, "ignore unbound environment variables")
	}
	opinionatedBindConfig(nil, hooks, &cmd, cfg, opts...)
	return &cmd
}

// RootGroup is a convenience function to construct a command group at the application's root.
// The command group does not take any special configuration.
func RootGroup(cmdTmpl cobra.Command, sub ...func(*cobra.Command)) *cobra.Command {
	cmd := RootCommand(Hooks[struct{}]{}, cmdTmpl, struct{}{})
	for _, subCmd := range sub {
		subCmd(cmd)
	}
	return cmd
}

func applyEnvToHelp(next func(cmd *cobra.Command, args []string)) func(cmd *cobra.Command, args []string) {
	return func(cmd *cobra.Command, args []string) {
		cmd.Flags().VisitAll(applyEnvToFlag(cmd, nil))
		next(cmd, args)
	}
}

func applyEnvToUsage(next func(cmd *cobra.Command) error) func(cmd *cobra.Command) error {
	return func(cmd *cobra.Command) error {
		cmd.Flags().VisitAll(applyEnvToFlag(cmd, nil))
		return next(cmd)
	}
}

// SubCommand wraps a cobra.Command to pass a config bound via BindConfig to a set of cobra run functions.
func SubCommand[T any](parent *cobra.Command, hooks Hooks[T], cmd cobra.Command, cfg T, opts ...Option) *cobra.Command {
	if parent == nil {
		panic("parent command must not be nil")
	}
	if Environment {
		hooks.PersistentPreRun = applyEnvToCmd(&cmd, hooks.PersistentPreRun)
	}
	opinionatedBindConfig(parent, hooks, &cmd, cfg, opts...)
	return &cmd
}

// SubGroup is a convenience function to construct a command group without special configuration.
func SubGroup(parent *cobra.Command, cmdTmpl cobra.Command, sub ...func(*cobra.Command)) *cobra.Command {
	cmd := SubCommand(parent, Hooks[struct{}]{}, cmdTmpl, struct{}{})
	for _, subCmd := range sub {
		subCmd(cmd)
	}
	return cmd
}

func opinionatedBindConfig[T any](parent *cobra.Command, hooks Hooks[T], cmd *cobra.Command, cfg T, opts ...Option) {
	// Local flags should just work, and the user is expected to provide a proper "Use" line for
	// the command that suggests where flags should go.
	if cmd.Use == "" {
		panic("use line must be set, and should include all non-global flags")
	}
	cmd.TraverseChildren = true
	cmd.DisableAutoGenTag = true
	cmd.DisableFlagsInUseLine = true

	// Accept no args unless those were explicitly allowed by the user.
	// pflag's default is to accept arbitrary args by default.
	if cmd.Args == nil {
		cmd.Args = cobra.NoArgs
	}

	// An option to register a parent is offered so that we know the full env path before binding
	// to the configuration structure.
	if parent != nil {
		parent.AddCommand(cmd)
	}

	// Disable flag sorting so that flags appear in the same order as in Go structs.
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false

	// Extend the parent command's prefix by default. This can be overridden via WithEnvPrefix.
	// Note we can't use cmd.CommandPath() here because it may return cmd.DisplayName().
	if Environment {
		var commands []string
		cmd.VisitParents(func(parent *cobra.Command) {
			commands = append([]string{parent.Name()}, commands...)
		})
		fullCommand := strings.Join(append(commands, cmd.Name()), " ")

		var envPrefix string
		if parent == nil {
			envPrefix = screamingSnake(cmd.Name())
		} else if parentPrefix, ok := parent.Annotations[annotationEnv]; ok {
			envPrefix = parentPrefix + screamingSnake(cmd.Name())
		} else {
			envPrefix = screamingSnake(fullCommand)
		}
		opts = append([]Option{WithEnvPrefix(envPrefix)}, opts...)

		cmd.AddCommand(newPrintEnvCmd(cmd, fullCommand))
	}

	cmd.PersistentPreRunE = passCfg(&cfg, hooks.PersistentPreRun)
	cmd.PreRunE = passCfg(&cfg, hooks.PreRun)
	cmd.RunE = passCfg(&cfg, hooks.Run)
	cmd.PostRunE = passCfg(&cfg, hooks.PostRun)
	cmd.PersistentPostRunE = passCfg(&cfg, hooks.PersistentPostRun)

	BindConfig(cmd, &cfg, opts...)
}

func passCfg[T any](cfg *T, f Hook[T]) func(cmd *cobra.Command, args []string) error {
	if f != nil {
		return func(cmd *cobra.Command, args []string) error {
			return f(cfg, cmd, args)
		}
	} else {
		return nil
	}
}
