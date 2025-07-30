nicecmd
=======

[Cobra](https://cobra.dev/) gives you nice CLIs already, and this package adds
nice bindings to your variables on top. It is for you if you want Cobra, but
programming with Cobra as-is seems just a bit too verbose for your taste.

* Define your config and defaults with structs
* This package uses reflection to pre-populate your `cobra.Command` with flags
* Environment variable and dotenv-style config file support is automatic

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
	cmd := nicecmd.RootCommand(nicecmd.Run(greet), cobra.Command{
		Use:   "nicecmd-readme --name <name> [-w <weather>]",
		Short: "It's just Cobra, but with no binding/setup required!",
	}, Config{
		Weather: "nice",
	})
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func greet(cfg *Config, cmd *cobra.Command, args []string) error {
	cmd.Printf("Hello, %s!\n", cfg.Name)
	cmd.Printf("The weather looks %s today!\n", cfg.Weather)
	return nil
}
```

```text
$ go run ./cmd/nicecmd-readme --help
It's just Cobra, but with no binding/setup required!

Usage:
  nicecmd-readme --name <name> [-w <weather>]
  nicecmd-readme [command]

Available Commands:
  completion  Generate the autocompletion script for the specified shell
  help        Help about any command
  printenv    Print all environment variable values or defaults for this command

Flags:
      --name string            person to greet (env NICECMD_README_NAME) (required)
  -w, --weather string         how's the weather? (env NICECMD_README_WEATHER) (default "nice")
      --env-file stringArray   load dotenv file (repeat for multiple files)
      --env-overwrite          give precedence to dotenv environment variables
      --env-lax                ignore unbound environment variables
  -h, --help                   help for nicecmd-readme

Use "nicecmd-readme [command] --help" for more information about a command.
```

A more complete example with a sub-command is available in [cmd/nicecmd-fizzbuzz](cmd/nicecmd-fizzbuzz).
Additionally, [reflect_test.go](reflect_test.go) and the documentation of
[Cobra](https://pkg.go.dev/github.com/spf13/cobra) and
[pflag](https://pkg.go.dev/github.com/spf13/pflag) will be useful for more complex CLI tools.

Principles and Patterns
-----------------------

### Immutable, type-safe, private configuration

All configuration is passed to commands as struct. Default values are encoded in a type-safe way in
the initial struct passed to `nicecmd.RootCommand`. Only the command's run function gets access to
the filled-out configuration.

### Avoid global variables

Cobra's examples suggest to have all commands in one package, and use Go's `init()` function to
register the command with some global `rootCmd`. I found that this quickly created clashes between
unrelated state of various subcommands. Likewise, you could access another command's variables,
despite them being uninitialized.

With `nicecmd` all configuration is in a struct, and you get a private copy of it to work with.
This avoids global variables for parameters.

### Sub-commands

`nicecmd.SubCommand` will prefix the command's env vars with the parent command's path.

You should structure sub-commands so that any shared configuration is a local (or persistent for
convenience) variable on the parent command. For example, a log level would be shared for the
entire application.

You however cannot access the configuration of a parent command in the sub-command! Instead, modify
the command context from the setup hook, e.g. to inject a logger:

```go
type RootConfig struct { LogLevel string }
type SubConfig struct {}

rootCmd := nicecmd.RootCommand(nicecmd.Setup(setup), cobra.Command{
	Use:   "foo [--log-level <level>] <command>"
	Short: "Foo will fizz your buzz"
}, RootConfig{})

nicecmd.SubCommand(rootCmd, nicecmd.Run(run), cobra.Command{
	Use:   "bar"
	Short: "Do the fizzing and buzzing"
}, SubConfig{})

func setup(cfg *RootConfig, cmd *cobra.Command, args []string) error {
	// This always gets called before bar (or any other sub-command).
	myLog := logutil.NewSLog(cfg.LogLevel)
	cmd.SetContext(logutil.WithLogContext(cmd.Context(), myLog))
}

func run(cfg *SubConfig, cmd *cobra.Command, args []string) error {
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
field name of your configuration structure. A prefix is derived from the first word of a command's
`Use` line. Individual field names can be overridden via `param` and `env`:

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
dependencies. NiceCmd does not care about configuration files: It gives you environment variables,
which is usually sufficient for configuring containerized applications.

Append `printenv` to a command to dump its configuration as dotenv file.

If you need more, you can set `nicecmd.Environment = false` and let Viper handle everything.

License
-------

NiceCmd is released under the Apache 2.0 license.

Contributions
-------------

I welcome contributions to this project, but may reject contributions that don't fit its spirit:

1. This is a rather opinionated library built on top of Cobra. It does not follow Cobra's defaults and conventions. If you need to customize something, then interacting with Cobra or changing its settings directly is usually the way to go.
2. The library should remain minimal. I will reject contributions that add dependencies. (The stdlib is mostly fine.)
3. I would treat this project like `go fmt`: If something looks off or is awkward to use, then that's a bug too. NiceCmd should after all make your command line code nice.
4. Contributions that come with code must come with tests.
5. Contributions must be licensed under the Apache 2.0 license.
