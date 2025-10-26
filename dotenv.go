package nicecmd

import (
	"fmt"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

// these may be replaced temporarily by tests
var (
	pkgDotEnvLoad     = godotenv.Load
	pkgDotEnvOverload = godotenv.Overload
)

func applyDotEnv[T any](rootCmd *cobra.Command, next Hook[T]) Hook[T] {
	fs := rootCmd.PersistentFlags()
	files := fs.StringArray("env-file", nil, "load dotenv file (repeat for multiple files)")
	overwrite := fs.Bool("env-overwrite", false, "give precedence to dotenv environment variables")
	return func(cfg *T, leafCmd *cobra.Command, args []string) error {
		if len(*files) != 0 {
			loadFunc := pkgDotEnvLoad
			if *overwrite {
				loadFunc = pkgDotEnvOverload
			}
			if err := loadFunc(*files...); err != nil {
				return fmt.Errorf("load dotenv: %s", err)
			}
		}
		if next != nil {
			return next(cfg, leafCmd, args)
		} else {
			return nil
		}
	}
}
