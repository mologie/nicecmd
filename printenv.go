package nicecmd

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func newPrintEnvCmd(outerCmd *cobra.Command, fullCommand string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "printenv",
		Short: "Print all environment variable values or defaults for this command",
		Args:  cobra.NoArgs,
	}
	cmd.DisableAutoGenTag = true
	cmd.DisableFlagsInUseLine = true

	//goland:noinspection GoUnhandledErrorResult for fmt.Fprintf
	printEnv := func(cmd *cobra.Command, args []string) {
		omitQuotes := regexp.MustCompile(`^[a-zA-Z0-9_]*$`)
		bashQuote := func(s string) string {
			// note this merely cosmetics for the generated env file, not for security
			if omitQuotes.MatchString(s) {
				return s
			} else {
				return strconv.Quote(s)
			}
		}

		w := cmd.OutOrStdout()
		fmt.Fprintf(w, "# %s\n", fullCommand)

		outerCmd.Flags().VisitAll(func(flag *pflag.Flag) {
			if flag.Hidden {
				return
			}

			annEnv := flag.Annotations[annotationEnv]
			if len(annEnv) == 0 {
				return
			}

			annUsage := flag.Annotations[annotationUsage]
			fmt.Fprintf(w, "\n# %s", flag.Name)
			if len(annUsage) > 0 {
				fmt.Fprintf(w, ": %s", annUsage[0])
			}
			if typeName := flag.Value.Type(); typeName != "" {
				fmt.Fprintf(w, " (type: %s)", typeName)
			}
			if flag.Deprecated != "" {
				fmt.Fprintf(w, " (deprecated: %s)", flag.Deprecated)
			}
			_, required := flag.Annotations[cobra.BashCompOneRequiredFlag]
			if required {
				fmt.Fprint(w, " (required)")
			}
			fmt.Fprintf(w, "\n")

			env := annEnv[0]
			if flag.Changed {
				fmt.Fprintf(w, "%s=%s\n", env, bashQuote(flag.Value.String()))
			} else {
				fmt.Fprintf(w, "# %s=%s\n", env, bashQuote(flag.DefValue))
			}
		})
	}

	cmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// The help func will be run before argument validation when either --help is given, Run is
		// absent, or ErrHelp is returned. However, --help won't call PreRunE. This is (ab)used
		// here to provide an env var dump and exit successfully before flag validation happens.
		cmd.SetHelpFunc(printEnv)
		return pflag.ErrHelp
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		// This func exists to mark this command as runnable and have Cobra call PreRunE.
		return nil
	}

	return cmd
}
