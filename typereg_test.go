package nicecmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"reflect"
	"testing"
)

func TestTypeReg_RegisterUnregister(t *testing.T) {
	type someType string
	RegisterType[someType](nil, nil)
	if reg, ok := typeRegs[reflect.TypeFor[someType]()]; !ok {
		t.Fatal("expected custom type to be registered")
	} else if reg.Name != "someType" {
		t.Fatalf("expected custom type name to be 'someType', got %q", reg.Name)
	}
	UnregisterType[someType]()
	if _, ok := typeRegs[reflect.TypeFor[someType]()]; ok {
		t.Fatal("expected custom type to be unregistered")
	}
}

func TestTypeReg_Reflect(t *testing.T) {
	type stringAlias string
	type Config struct {
		Foo stringAlias
	}

	// test that stringAlias is not recognized prior to registration
	expectPanic(t, "unsupported field type", func() {
		cmd := &cobra.Command{Use: "nicecmd-test"}
		cfg := &Config{}
		BindConfig(cmd, cfg)
	})

	// test that Foo's value can be set after registration
	defer UnregisterType[stringAlias]()
	RegisterType[stringAlias](
		func(value string) (stringAlias, error) { return stringAlias(value), nil },
		func(value stringAlias) string { return string(value) },
	)
	cmd := &cobra.Command{Use: "nicecmd-test"}
	cfg := &Config{}
	BindConfig(cmd, cfg)
	cmd.SetArgs([]string{"--foo", "foo"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if cfg.Foo != "foo" {
		t.Fatalf(`expected cfg.Foo to be "foo", got %q`, cfg.Foo)
	}
}

var _ = func() pflag.Value {
	var result thirdPartyFlag
	return &result
}()

type thirdPartyFlag string

func (f *thirdPartyFlag) Set(value string) error {
	// if this setter is called, then our sentinel becomes "foo-pflag" and the test would fail
	*f = thirdPartyFlag(value + "-pflag")
	return nil
}

func (f *thirdPartyFlag) String() string {
	return string(*f)
}

func (f *thirdPartyFlag) Type() string {
	return "thirdPartyFlag"
}

func TestTypeReg_ReflectPrecedence(t *testing.T) {
	type Config struct {
		Foo thirdPartyFlag
	}

	pflagCmd := &cobra.Command{Use: "nicecmd-test"}
	pflagCfg := &Config{}
	BindConfig(pflagCmd, pflagCfg)
	pflagCmd.SetArgs([]string{"--foo", "foo"})
	if err := pflagCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if pflagCfg.Foo != "foo-pflag" {
		t.Fatalf(`expected pflagCfg.Foo to be "foo-pflag", got %q`, pflagCfg.Foo)
	}

	defer UnregisterType[thirdPartyFlag]()
	RegisterType[thirdPartyFlag](
		func(value string) (thirdPartyFlag, error) { return thirdPartyFlag(value + "-cust"), nil },
		func(value thirdPartyFlag) string { return string(value) },
	)
	customCmd := &cobra.Command{Use: "nicecmd-test"}
	customCfg := &Config{}
	BindConfig(customCmd, customCfg)
	customCmd.SetArgs([]string{"--foo", "foo"})
	if err := customCmd.Execute(); err != nil {
		t.Fatalf("expected no error, got %q", err)
	}
	if customCfg.Foo != "foo-cust" {
		t.Fatalf(`expected customCfg.Foo to be "foo-cust", got %q`, customCfg.Foo)
	}
}
