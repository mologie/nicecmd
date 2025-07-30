package nicecmd

import (
	"bytes"
	"github.com/spf13/cobra"
	"testing"
	"time"
)

func TestNewPrintEnvCmd_Nested(t *testing.T) {
	type RootConfig struct {
		Foo   int     `usage:"an integer"` // will not appear in printenv, relevant for root cmd's run only
		Bar   float64 `flag:"persistent" usage:"a float, persistent"`
		Delta time.Duration
	}
	runRoot := func(cfg *RootConfig, cmd *cobra.Command, args []string) error {
		return nil
	}
	rootCmd := RootCommand(Run(runRoot), cobra.Command{Use: "nicecmd-test"}, RootConfig{
		Delta: 5 * time.Minute,
	})

	type SubConfig struct {
		Baz   string `usage:"a string"`
		Gamma time.Duration
	}
	runSub := func(cfg *SubConfig, cmd *cobra.Command, args []string) error {
		return nil
	}
	SubCommand(rootCmd, Run(runSub), cobra.Command{Use: "sub"}, SubConfig{
		Gamma: 10 * time.Minute,
	})

	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"--foo=1", "sub", "printenv", "--bar=42"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	const expected = `# nicecmd-test sub

# baz: a string (type: string)
# NICECMD_TEST_SUB_BAZ=

# gamma (type: duration)
# NICECMD_TEST_SUB_GAMMA=10m0s

# bar: a float, persistent (type: float64)
NICECMD_TEST_BAR=42
`
	if actual := buf.String(); actual != expected {
		t.Errorf("output mismatch, actual: %v", actual)
	}
}

func TestNewPrintEnvCmd_Required(t *testing.T) {
	type Config struct {
		Required    bool `flag:"required"`
		NotRequired int
	}
	run := func(cfg *Config, cmd *cobra.Command, args []string) error {
		return nil
	}
	cmd := RootCommand(Run(run), cobra.Command{Use: "nicecmd-test"}, Config{})
	if err := cmd.Flags().Set("not-required", "42"); err != nil {
		t.Fatalf(`set flag "not-required": %v`, err)
	}
	buf := &bytes.Buffer{}
	cmd.SetOut(buf)
	cmd.SetArgs([]string{"printenv"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
	const expected = `# nicecmd-test

# required (type: bool) (required)
# NICECMD_TEST_REQUIRED=false

# not-required (type: int)
NICECMD_TEST_NOT_REQUIRED=42
`
	if actual := buf.String(); actual != expected {
		t.Errorf("output mismatch, actual: %v", actual)
	}
}
