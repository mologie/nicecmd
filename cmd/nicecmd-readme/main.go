package main

import (
	"github.com/mologie/nicecmd"
	"github.com/spf13/cobra"
	"os"
)

type Config struct {
	Name    string `usage:"your name"`
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
