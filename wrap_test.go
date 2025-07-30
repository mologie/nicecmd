package nicecmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"github.com/spf13/cobra"
	"os"
	"reflect"
	"strconv"
	"strings"
	"testing"
)

type trivialConf struct {
	Foo string
	Bar int
}

type errFooMismatch struct {
	Actual string
}

func (e errFooMismatch) Error() string {
	return fmt.Sprintf(`expected cfg.Foo="foo", got %q`, e.Actual)
}

func trivialRun(cfg *trivialConf, cmd *cobra.Command, args []string) error {
	if cfg.Foo == "foo" {
		return nil
	} else {
		return errFooMismatch{Actual: cfg.Foo}
	}
}

func TestWrap_Execute(t *testing.T) {
	cmd := RootCommand(Run(trivialRun), cobra.Command{Use: "nicecmd-test"}, trivialConf{})
	if reflect.ValueOf(cmd.Args).Pointer() != reflect.ValueOf(cobra.NoArgs).Pointer() {
		t.Error("expected cmd to accept no args")
	}
	cmd.SetArgs([]string{"--foo", "foo"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
}

func TestWrap_Groups(t *testing.T) {
	var subs []func(*cobra.Command)
	for i := 1; i <= 3; i++ {
		subs = append(subs, func(parent *cobra.Command) {
			cmdTmpl := cobra.Command{Use: fmt.Sprintf("sub%d", i)}
			SubGroup(parent, cmdTmpl, func(parent *cobra.Command) {
				SubCommand(parent, Run(trivialRun), cobra.Command{Use: "leaf"}, trivialConf{})
			})
		})
	}
	cmd := RootGroup(cobra.Command{Use: "nicecmd-test"}, subs...)
	subCmds := cmd.Commands()
	ourCmds := 0
	for _, subCmd := range subCmds {
		if strings.HasPrefix(subCmd.Name(), "sub") {
			ourCmds++
		}
	}
	if ourCmds != 3 {
		t.Errorf("expected 3 new sub-commands to be created, got %d", ourCmds)
	}
	cmd.SetArgs([]string{"sub1", "leaf", "--foo", "foo"})
	if err := cmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
}

func TestWrap_EnvironmentFeaturesDisabled(t *testing.T) {
	if !Environment {
		t.Fatal("environment features should be enabled by default")
	}
	defer func() { Environment = true }()
	Environment = false
	defer tempEnv(t, [][2]string{{"NICECMD_TEST_FOO", "foo"}})()
	cmd := RootCommand(Run(trivialRun), cobra.Command{Use: "nicecmd-test"}, trivialConf{})
	if cmd.Flags().Lookup("env-file") != nil {
		t.Error("expected env-file flag to be absent")
	}
	if cmd.Flags().Lookup("env-override") != nil {
		t.Error("expected env-override flag to be absent")
	}
	if cmd.Flags().Lookup("env-lax") != nil {
		t.Error("expected env-lax flag to be absent")
	}
	cmd.SetArgs([]string{})
	var errFoo errFooMismatch
	if err := cmd.Execute(); !errors.As(err, &errFoo) {
		t.Errorf("expected cmd to fail with errFooMismatch, got: %T", err)
	} else if errFoo.Actual != "" {
		t.Errorf("expected cmd to not receive a value for --foo, got: %v", errFoo.Actual)
	}
}

func TestWrap_SubContextPropagation(t *testing.T) {
	type contextKey struct{}

	rootOtherCalled := 0
	rootSetup := func(cfg *trivialConf, cmd *cobra.Command, args []string) error {
		cmd.SetContext(context.WithValue(cmd.Context(), contextKey{}, cfg.Foo))
		return nil
	}
	rootOther := func(cfg *trivialConf, cmd *cobra.Command, args []string) error {
		rootOtherCalled++
		return nil
	}
	rootCmd := RootCommand(Hooks[trivialConf]{
		PersistentPreRun:  rootSetup, // called
		PreRun:            rootOther,
		Run:               rootOther,
		PostRun:           rootOther,
		PersistentPostRun: rootOther, // called
	}, cobra.Command{
		Use: "root",
	}, trivialConf{
		Foo: "foo default",
	})

	type SubConf struct {
		Bar string
	}
	subOtherCalled := 0
	subOther := func(cfg *SubConf, cmd *cobra.Command, args []string) error {
		subOtherCalled++
		return nil
	}
	subMain := func(cfg *SubConf, cmd *cobra.Command, args []string) error {
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
		PersistentPreRun:  subOther,
		PreRun:            subOther,
		Run:               subMain,
		PostRun:           subOther,
		PersistentPostRun: subOther,
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
	if rootOtherCalled != 1 {
		t.Errorf("expected one other hook to be called on root command, got %d", rootOtherCalled)
	}
	if subOtherCalled != 4 {
		t.Errorf("expected 4 other hooks to be called on sub-command, got %d", subOtherCalled)
	}
}

type testSetCounter int

func (v *testSetCounter) String() string {
	return strconv.Itoa(int(*v))
}

func (v *testSetCounter) Set(string) error {
	*v++
	return nil
}

func (v *testSetCounter) Type() string {
	return "testSetCounter"
}

func TestWrap_SubEnvAppliedOnce(t *testing.T) {
	type CountConfig struct {
		Count testSetCounter
	}
	nop := func(cfg *CountConfig, cmd *cobra.Command, args []string) error {
		return nil
	}
	check := func(cfg *CountConfig, cmd *cobra.Command, args []string) error {
		if cfg.Count != 1 {
			return fmt.Errorf(`expected one set call, but got %d`, cfg.Count)
		}
		return nil
	}
	defer tempEnv(t, [][2]string{{"NICECMD_TEST_COUNT", "blubi"}})()
	rootCmd := RootCommand(Hooks[CountConfig]{
		Run:               nop,
		PersistentPostRun: check,
	}, cobra.Command{Use: "nicecmd-test"}, CountConfig{})
	SubCommand(rootCmd, Run(nop), cobra.Command{Use: "sub"}, CountConfig{})
	rootCmd.SetArgs([]string{"sub"})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
}

func TestWrap_SubEnvVars(t *testing.T) {
	checksOK := 0
	checkBar := func(i int) Hook[trivialConf] {
		return func(cfg *trivialConf, cmd *cobra.Command, args []string) error {
			if cfg.Bar != i {
				return fmt.Errorf(`expected --bar=%d, got %d`, i, cfg.Bar)
			}
			checksOK++
			return nil
		}
	}
	failOnRun := func(cfg *trivialConf, cmd *cobra.Command, args []string) error {
		return errors.New("this should not be called")
	}

	// build tree: root -> sub1 -> sub2 -> sub3 -> leaf
	// root wants --foo=foo
	// each sub command wants --bar=i
	rootCmd := RootCommand(
		Setup(trivialRun),
		cobra.Command{Use: "nicecmd-test"},
		trivialConf{},
		WithEnvPrefix("NICECMD_CUSTOM"))
	nextCmd := rootCmd
	for i := 1; i <= 3; i++ {
		nextCmd = SubCommand(nextCmd, SetupAndRun(checkBar(i), failOnRun), cobra.Command{
			Use: fmt.Sprintf("sub%d", i),
		}, trivialConf{})
	}
	SubCommand(nextCmd, SetupAndRun(checkBar(4), trivialRun), cobra.Command{
		Use: "leaf",
	}, trivialConf{})

	defer tempEnv(t, [][2]string{
		{"NICECMD_CUSTOM_FOO", "foo"},
		{"NICECMD_CUSTOM_SUB1_BAR", "1"},
		{"NICECMD_CUSTOM_SUB1_SUB2_BAR", "2"},
		{"NICECMD_CUSTOM_SUB1_SUB2_SUB3_BAR", "3"},
		{"NICECMD_CUSTOM_SUB1_SUB2_SUB3_LEAF_FOO", "foo"},
		{"NICECMD_CUSTOM_SUB1_SUB2_SUB3_LEAF_BAR", "4"},
	})()
	rootCmd.SetArgs([]string{"sub1", "sub2", "sub3", "leaf"})
	if err := rootCmd.Execute(); err != nil {
		t.Errorf("execute: %v", err)
	}
	if checksOK != 4 {
		t.Errorf("expected 4 checks to pass, got %d", checksOK)
	}
}

func TestWrap_DotEnv(t *testing.T) {
	tt := []struct {
		name      string
		overwrite bool
	}{
		{"without overwrite", false},
		{"with overwrite", true},
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"--env-file", ".env", "sub"}
			if tc.overwrite {
				args = append(args, "--env-overwrite")
			}
			loaded := false
			f := &pkgDotEnvLoad
			if tc.overwrite {
				f = &pkgDotEnvOverload
			}
			defer replace(f, func(filenames ...string) error {
				if len(filenames) == 0 || filenames[0] != ".env" {
					return fmt.Errorf("expected filename .env, got %q", filenames)
				}
				_ = os.Setenv("NICECMD_TEST_FOO", "foo")
				_ = os.Setenv("NICECMD_TEST_SUB_FOO", "foo")
				loaded = true
				return nil
			})()
			defer func() {
				_ = os.Unsetenv("NICECMD_TEST_FOO")
				_ = os.Unsetenv("NICECMD_TEST_SUB_FOO")
			}()
			rootCmd := RootCommand(Setup(trivialRun), cobra.Command{Use: "nicecmd-test"}, trivialConf{})
			rootCmd.SetArgs(args)
			SubCommand(rootCmd, Run(trivialRun), cobra.Command{Use: "sub"}, trivialConf{})
			if err := rootCmd.Execute(); err != nil {
				t.Errorf("execute: %v", err)
			}
			if !loaded {
				t.Error("expected .env file to be loaded, but it was not")
			}
		})
	}
}

func TestWrap_MissingUsage(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic")
		} else if !strings.Contains(r.(string), "use line") {
			t.Errorf("unexpected panic: %v", r)
		}
	}()
	RootCommand(Run(trivialRun), cobra.Command{}, trivialConf{})
}

func TestWrap_UsageAndExitOnBadConfig(t *testing.T) {
	defer tempEnv(t, [][2]string{{"NICECMD_TEST_BAR", "not-an-integer"}})()
	buf := &bytes.Buffer{}
	cmd := RootCommand(Run(trivialRun), cobra.Command{Use: "nicecmd-test"}, trivialConf{})
	cmd.SetOut(buf)
	var errInvalid ErrInvalidEnvironment
	if err := cmd.Execute(); !errors.As(err, &errInvalid) {
		t.Errorf("expected Command to fail with ErrInvalidEnvironment, got: %v", err)
	}
	if len(errInvalid.FlagErrors) != 1 || errInvalid.FlagErrors[0].Flag.Name != "bar" {
		t.Errorf("expected Command to fail due to flag bar")
	}
	if out := buf.String(); !strings.Contains(out, "Usage:") {
		t.Errorf("expected Command to print usage on invalid env, but got output: %v", out)
	}
}

func TestWrap_UnboundEnv(t *testing.T) {
	tt := []struct {
		name string
		args []string
		num  int
	}{
		{"root command", []string{}, 3},     // one in root, both in unused sub
		{"sub command", []string{"sub"}, 2}, // one in root, one in sub
	}
	for _, tc := range tt {
		t.Run(tc.name, func(t *testing.T) {
			defer tempEnv(t, [][2]string{
				{"NICECMD_TEST_FOO", "foo"},
				{"NICECMD_TEST_UNBOUND", "bar"},
				{"NICECMD_TEST_SUB_FOO", "foo"},
				{"NICECMD_TEST_SUB_UNBOUND", "bar"},
			})()
			rootCmd := RootCommand(Run(trivialRun), cobra.Command{Use: "nicecmd-test"}, trivialConf{})
			rootCmd.SetArgs(tc.args)
			SubCommand(rootCmd, Run(trivialRun), cobra.Command{Use: "sub"}, trivialConf{})
			var errUnbound ErrUnboundEnvironment
			if err := rootCmd.Execute(); !errors.As(err, &errUnbound) {
				t.Errorf("expected Command to fail with ErrUnboundEnvironment, got: %v", err)
			}
			if len(errUnbound.Names) != tc.num {
				t.Errorf("expected Command to fail with %d unbound env vars, got: %v", tc.num, errUnbound.Names)
			}
		})
	}
}

func tempEnv(t *testing.T, envs [][2]string) func() {
	t.Helper()
	for _, env := range envs {
		if err := os.Setenv(env[0], env[1]); err != nil {
			t.Fatalf("failed to set env %s=%s: %v", env[0], env[1], err)
		}
	}
	return func() {
		for _, env := range envs {
			if err := os.Unsetenv(env[0]); err != nil {
				t.Errorf("failed to unset env %s=%s: %v", env[0], env[1], err)
			}
		}
	}
}

func replace[T any](p *T, hook T) func() {
	old := *p
	*p = hook
	return func() {
		*p = old
	}
}
