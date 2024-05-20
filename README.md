nicecmd
=======

[Cobra](https://cobra.dev/) gives you nice CLIs already, and this package adds
nice bindings to your variables on top. It is for you if you want Cobra, but
programming with Cobra as-is seems just a bit too verbose for your taste.

* Define your config and defaults with structs
* This package uses reflection to pre-populate your `cobra.Command` with flags
* Values are pre-set from environment variables before reading the command line
* Supports all `spf13/pflag` (Cobra flag package) types, plus your own
* Nothing is hidden, you're free to customize your commands, e.g. with Viper

```go
package main

import (
	"github.com/mologie/nicecmd"
	"github.com/spf13/cobra"
	"os"
)

type Config struct {
	Name    string `flag:"required" usage:"person to greet"`
	Weather string `param:"w" usage:"how's the weather?"`
}

func main() {
	cmd := nicecmd.Command("HELLO", nicecmd.Run(greet), cobra.Command{
		Use:   "nicecmd-example --name <name> [-w <weather>]",
		Short: "It's just Cobra, but with no binding/setup required!",
	}, Config{
		Weather: "nice",
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func greet(cfg Config, cmd *cobra.Command, args []string) error {
	cmd.Printf("Hello, %s!\n", cfg.Name)
	cmd.Printf("The weather looks %s today!\n", cfg.Weather)
	return nil
}
```

```text
$ go run ./cmd/nicecmd-readme
Error: required flag(s) "name" not set
Usage:
  nicecmd-example --name <name> [-w <weather>]

Flags:
  -h, --help             help for nicecmd-example
      --name string      person to greet (required) (env HELLO_NAME)
  -w, --weather string   how's the weather? (env HELLO_WEATHER) (default "nice")
```

A more complete example with a sub-command is available in [cmd/nicecmd-fizzbuzz](cmd/nicecmd-fizzbuzz).
Additionally, [reflect_test.go](reflect_test.go) and the documentation of
[Cobra](https://pkg.go.dev/github.com/spf13/cobra) and
[pflag](https://pkg.go.dev/github.com/spf13/pflag) will be useful for more complex CLI tools.

Principles and Patterns
-----------------------

### Immutable, type-safe, private configuration

All configuration is passed to commands by value as copy. Default values are encoded in a type-safe
way in the initial struct passed to `nicecmd.Command`. Only the command's run function gets access
to the filled-out configuration.

### Avoid global variables

Cobra's examples suggest to have all commands in one package, and use Go's `init()` function to
register the command with some global `rootCmd`. I found that this quickly created clashes between
unrelated state of various subcommands. Likewise, you could access another command's variables,
despite them being uninitialized.

With `nicecmd` all configuration is in a struct, and you get an immutable copy of it to work with.
This avoids global variables for parameters.

You can further avoid having a global (sub)command variables by consolidating all `cmd.AddCommand`
calls in `main`, or separate per-command-package `NewCommand` methods, whatever floats your boat.

### Sub-commands

Use `AddCommand` on any `cobra.Command`, regardless of whether it was created through nicecmd or
directly through Cobra. However, note that nicecmd will:

* Set `EnableTraverseRunHooks`: Persistent pre-run hooks of parents are always run
* Set `TraverseChildren`: Parameters of the config struct passed to such hooks are set
* Set `DisableFlagsInUseLine`: Your `Use` line will appear as-is in docs

You should structure sub-commands so that any shared configuration is a local (or persistent for
convenience) variable on the parent command. For example, a log level would be shared for the
entire application.

You however cannot access the configuration of a parent command in the sub-command! Instead, modify
the command context from the pre-run hook, e.g. to inject a logger:

```go
type RootConfig struct { LogLevel string }
type SubConfig struct {}

rootCmd := nicecmd.Command("FOO", nicecmd.PersistentPreRun(setup), cobra.Command{
	Use:   "foo [--log-level <level>] <command>"
	Short: "Foo will fizz your buzz"
}, RootConfig{})

rootCmd.AddCommand(&nicecmd.Command("FOO_BAR", nicecmd.Run(run), cobra.Command{
	Use:   "bar"
	Short: "Do the fizzing and buzzing"
}, SubConfig{}))

func setup(cfg RootConfig, cmd *cobra.Command, args []string) error {
	// This always gets called before bar (or any other sub-command).
	myLog := logutil.NewSLog(cfg.LogLevel)
	cmd.SetContext(logutil.WithLogContext(cmd.Context(), myLog))
}

func run(cfg SubConfig, cmd *cobra.Command, args []string) error {
	log := logutil.FromContext(cmd.Context())
	log.Debug("fizz buzzing will commence") // but is omitted
}
```

This pattern should apply to pretty much any kind of state that you need to create and inherit to
sub-commands. If you need an escape hatch, you can still update the context with a pointer to the
entire `RootConfig` struct and let your sub-command do the setup regardless.

### Required parameters

Use `flag:"required"` to mark a flag as required. This is preferred over checking for absent values
in code, because Cobra will aggregate errors and display all missing flags to the user for you.

### Persistent parameters

Cobra has a concept of persistent parameters. A flag can be made persistent via `flag:"persistent"`.
This tag is comma-separated, e.g. `flag:"required,persistent"` is valid.

To illustrate with an example an optional persistent `--log-level` on the root command would make
both of these invocations valid:

* `foo --log-level=debug server`
* `foo server --log-level=debug`

Whereas if `log-level` was not persistent, only the first command would work.

### Automatic naming

This package will automatically derive a name for parameters and environment variables from the
field name of your configuration structure. An optional prefix for environment variables (`HELLO_`
in the example above) can be set. Names can be overridden via `param` and `env`:

* `FooBarBaz string` is set via param `--foo-bar-baz` or env var `FOO_BAR_BAZ`
* Use `param:"foo"` to change just the long form
* Use `param:"foo,f"` to change the long form and add a short form
* Use `param:"f"` to add a short form and keep the default long name
* Every parameter must have a long form, I find that more intuitive.
* Use `env:"FOO"` to define a custom environment variable to read from. No prefix will be added!
* Use `env:"-"` to remove the environment variable. Useful for flags like `--version`.

### Sub-structs are flattened with a prefix

Take the following example, where `Config` is used for some `nicecmd.Command`:

```go
type LogConfig struct {
	Level  int    `usage:"raise the bar"`
	Format string `usage:"TEXT or JSON"`
}

type Config struct {
	Log LogConfig `flag:"persistent"`
}
```

* This gets you the parameters `--log-level` and `--log-format`.
* Using `param` on `Log` would change the prefix.
* Flag options are inherited: The whole struct becomes persistent.

### Configuration files

Viper (from the authors of Cobra) is a pretty nice configuration library, but comes with a bunch of
dependencies. NiceCmd does not care about configuration at all: It gives you environment variables,
which is usually sufficient for configuring containerized applications.

If you need more, you can set `nicecmd.Environment = false` and let Viper do the work. 

License
-------

NiceCmd is released under the Apache 2.0 license.

Contributions
-------------

I welcome contributions to this project, but may reject contributions that don't fit its spirit:

1. This is a rather opinionated library that build on top of Cobra. It does not follow Cobra defaults exactly. If you just need to customize something, then interacting with Cobra or changing its settings directly is usually the way to go.
2. The library should remain minimal. I will reject contributions that add dependencies. (The stdlib is mostly fine.)
3. I would treat this project like `go fmt`: If something looks off or is awkward to use, then that's a bug too. NiceCmd should after all make your command line code nice.
4. Contributions that come with code must come with tests.
5. Contributions must be licensed under the Apache 2.0 license.
