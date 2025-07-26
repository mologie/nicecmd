package nicecmd

import (
	"fmt"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"maps"
	"os"
	"slices"
	"strings"
)

type ErrUnboundEnvironment struct {
	Names []string
}

func (e ErrUnboundEnvironment) Error() string {
	var sb strings.Builder
	sb.WriteString("unbound environment variables:\n")
	for _, name := range e.Names {
		sb.WriteString(fmt.Sprintf("  %s\n", name))
	}
	return sb.String()
}

func checkEnv[T any](rootCmd *cobra.Command, next Hook[T]) Hook[T] {
	return func(cfg *T, leafCmd *cobra.Command, args []string) error {
		if lax, err := rootCmd.Flags().GetBool("env-lax"); err != nil {
			panic("error reading env-lax flag state: " + err.Error())
		} else if !lax {
			prefix := rootCmd.Annotations[annotationEnv]
			unbound := make(map[string]struct{})
			for _, env := range os.Environ() {
				key, _, ok := strings.Cut(env, "=")
				if ok && strings.HasPrefix(key, prefix) {
					unbound[key] = struct{}{}
				}
			}
			prune := func(flag *pflag.Flag) {
				if annotations, ok := flag.Annotations[annotationEnv]; ok {
					delete(unbound, annotations[0])
				}
			}
			leafCmd.LocalFlags().VisitAll(prune)
			leafCmd.VisitParents(func(cmd *cobra.Command) {
				cmd.LocalFlags().VisitAll(prune)
			})
			if len(unbound) > 0 {
				return ErrUnboundEnvironment{Names: slices.Sorted(maps.Keys(unbound))}
			}
		}
		if next != nil {
			return next(cfg, leafCmd, args)
		} else {
			return nil
		}
	}
}
