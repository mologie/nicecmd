package nicecmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"os"
	"strings"
)

const (
	annotationProcessed = "nicecmd_processed"
	annotationUsage     = "nicecmd_usage"
)

type FlagError struct {
	Flag  *pflag.Flag
	Error error
}

type ErrInvalidEnvironment struct {
	FlagErrors []FlagError
}

func (e ErrInvalidEnvironment) Error() string {
	var sb strings.Builder
	sb.WriteString("invalid environment variables:\n")
	for _, flag := range e.FlagErrors {
		sb.WriteString(fmt.Sprintf("  %s: %s\n", flag.Flag.Name, flag.Error))
	}
	return sb.String()
}

func applyEnvToCmd[T any](cmd *cobra.Command, next Hook[T]) Hook[T] {
	return func(cfg *T, leafCmd *cobra.Command, args []string) error {
		var err ErrInvalidEnvironment
		cmd.LocalFlags().VisitAll(applyEnvToFlag(cmd, &err.FlagErrors))
		if len(err.FlagErrors) > 0 {
			return err
		} else if next != nil {
			return next(cfg, leafCmd, args)
		} else {
			return nil
		}
	}
}

func applyEnvToFlag(cmd *cobra.Command, errors *[]FlagError) func(*pflag.Flag) {
	return func(flag *pflag.Flag) {
		if _, ok := flag.Annotations[annotationProcessed]; ok {
			return
		}
		if flag.Annotations == nil {
			flag.Annotations = make(map[string][]string)
		}
		if len(flag.Usage) != 0 {
			flag.Annotations[annotationUsage] = []string{flag.Usage} // backup for printenv
		}
		if annotations, ok := flag.Annotations[annotationEnv]; ok {
			env := annotations[0]
			if value, ok := os.LookupEnv(env); ok {
				ansiColor := "32" // green
				if err := flag.Value.Set(value); err != nil {
					cmd.Printf("Error: environment variable %q: %s\n", env, err)
					ansiColor = "31" // red
					if errors != nil {
						*errors = append(*errors, FlagError{
							Flag:  flag,
							Error: err,
						})
					}
				} else {
					flag.Changed = true
				}
				spaceAppendf(&flag.Usage, "(\033[%smenv %s=%q\033[0m)", ansiColor, env, value)
			} else {
				spaceAppendf(&flag.Usage, "(env %s)", env)
			}
			flag.Annotations[annotationProcessed] = []string{}
		}
		if _, ok := flag.Annotations[cobra.BashCompOneRequiredFlag]; ok {
			spaceAppend(&flag.Usage, "(required)")
		}
	}
}

func spaceAppend(s *string, suffix string) {
	if len(*s) > 0 {
		*s += " "
	}
	*s += suffix
}

func spaceAppendf(s *string, format string, a ...any) {
	spaceAppend(s, fmt.Sprintf(format, a...))
}
