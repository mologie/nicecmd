package nicecmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"regexp"
	"strconv"
)

func newPrintEnvCmd(outerCmd *cobra.Command, fullCommand string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "printenv",
		Short: "Print all environment variable values or defaults for this command",
		Args:  cobra.NoArgs,
	}
	cmd.DisableAutoGenTag = true
	cmd.DisableFlagsInUseLine = true
	cmd.RunE = func(*cobra.Command, []string) error {
		omitQuotes := regexp.MustCompile(`^[a-zA-Z0-9_]+$`)
		bashQuote := func(s string) string {
			// note this merely cosmetics for the generated env file, not for security
			if omitQuotes.MatchString(s) {
				return s
			} else {
				return strconv.Quote(s)
			}
		}

		if cmd.HasFlags() {
			_, _ = fmt.Fprintf(os.Stdout, "# %s\n", fullCommand)
			outerCmd.Flags().VisitAll(func(flag *pflag.Flag) {
				if flag.Hidden {
					return
				}

				annEnv := flag.Annotations[annotationEnv]
				if len(annEnv) == 0 {
					return
				}

				annUsage := flag.Annotations[annotationUsage]
				_, _ = fmt.Fprintf(os.Stdout, "\n# %s", flag.Name)
				if len(annUsage) > 0 {
					_, _ = fmt.Fprintf(os.Stdout, ": %s", annUsage[0])
				}
				if typeName := flag.Value.Type(); typeName != "" {
					_, _ = fmt.Fprintf(os.Stdout, " (type: %s)", typeName)
				}
				if flag.Deprecated != "" {
					_, _ = fmt.Fprintf(os.Stdout, " (deprecated: %s)", flag.Deprecated)
				}
				_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
				if required {
					_, _ = os.Stdout.WriteString(" (required)")
				}
				_, _ = os.Stdout.WriteString("\n")

				env := annEnv[0]
				if flag.Changed {
					_, _ = fmt.Fprintf(os.Stdout, "%s=%s\n", env, bashQuote(flag.Value.String()))
				} else {
					_, _ = fmt.Fprintf(os.Stdout, "# %s=%s\n", env, bashQuote(flag.DefValue))
				}
			})
		}
		return nil
	}
	return cmd
}
