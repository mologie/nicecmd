package nicecmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"strings"
	"testing"
)

type TrivialConf struct {
	Foo string
	Bar []int
}

func trivialRun(cfg TrivialConf, cmd *cobra.Command, args []string) error {
	if cfg.Foo == "foo" {
		return nil
	} else {
		return fmt.Errorf(`expected cfg.Foo="foo", got %q`, cfg.Foo)
	}
}

func TestCommand_Execute(t *testing.T) {
	cmd := RootCommand(Run(trivialRun), cobra.Command{Use: "test"}, TrivialConf{})

	if reflect.ValueOf(cmd.Args).Pointer() != reflect.ValueOf(cobra.NoArgs).Pointer() {
		t.Errorf("expected cmd to accept no args, to validator %p", cmd.Args)
	}

	cmd.SetArgs([]string{"--foo", "foo"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
}

func TestCommand_Sub(t *testing.T) {
	type contextKey struct{}

	rootMain := func(cfg TrivialConf, cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), contextKey{}, cfg.Foo))
		return nil
	}
	rootCmd := RootCommand(Setup(rootMain), cobra.Command{
		Use: "root",
	}, TrivialConf{
		Foo: "foo default",
	})

	type SubConf struct {
		Bar string
	}
	subHooksCalled := 0
	subHook := func(cfg SubConf, cmd *cobra.Command, args []string) error {
		subHooksCalled++
		return nil
	}
	subMain := func(cfg SubConf, cmd *cobra.Command, args []string) error {
		if foo, ok := cmd.Context().Value(contextKey{}).(string); !ok || foo != "foo" {
			return errors.New("parent state did not propagate via context")
		} else if len(args) != 1 || args[0] != "baz" {
			return errors.New("did not get my baz as only arg")
		} else if cfg.Bar != "bar" {
			return fmt.Errorf(`expected non-default value --bar=bar, but got %q`, cfg.Bar)
		} else {
			return nil
		}
	}
	subCmd := SubCommand(rootCmd, Hooks[SubConf]{
		PersistentPreRun:  subHook,
		PreRun:            subHook,
		Run:               subMain,
		PostRun:           subHook,
		PersistentPostRun: subHook,
	}, cobra.Command{
		Use:  "sub",
		Args: cobra.ArbitraryArgs,
	}, SubConf{
		Bar: "bar default",
	})

	subUsage := subCmd.UsageString()
	if !strings.Contains(subUsage, "env ROOT_SUB_BAR") {
		t.Errorf("expected sub-command usage to reference ROOT_SUB_BAR env var, got: %s", subUsage)
	}

	rootCmd.SetArgs([]string{"--foo", "foo", "sub", "--bar", "bar", "baz"})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
	if subHooksCalled != 4 {
		t.Errorf("expected 4 hooks to be called on sub-command, got %d", subHooksCalled)
	}
}

func TestCommand_MissingUsage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		} else if !strings.Contains(r.(string), "use line") {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	RootCommand(Run(trivialRun), cobra.Command{}, TrivialConf{})
}

func TestCommand_UsageAndExitOnBadConfig(t *testing.T) {
	exitCalled := false
	osExitOrTestHook = func(code int) {
		exitCalled = true
	}
	defer func() { osExitOrTestHook = os.Exit }()

	type EnvConfig struct {
		Bad int
	}
	if err := os.Setenv("NICECMD_TESTCMD_BAD", "value"); err != nil {
		t.Errorf("setenv: %v", err)
		return
	}

	buf := &bytes.Buffer{}
	cmdTemplate := cobra.Command{Use: "nicecmd-testcmd"}
	cmdTemplate.SetOut(buf)
	cmd := RootCommand(Hooks[EnvConfig]{}, cmdTemplate, EnvConfig{})
	if cmd != nil {
		t.Error("expected Command to fail")
	}
	if !exitCalled {
		t.Error("expected os.Exit to be called")
	}
	if out := buf.String(); !strings.Contains(out, "Usage:") {
		t.Errorf("expected Command to print usage on invalid env, but got output: %v", out)
	}
}
