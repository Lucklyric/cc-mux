package cli

import (
	"fmt"
	"sort"
	"text/tabwriter"

	"github.com/Lucklyric/cc-mux/internal/config"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List profiles and their descriptions",
		Args:    cobra.NoArgs,
		Example: "  cc-mux list",
		RunE: func(cmd *cobra.Command, _ []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			names := cfg.ProfileNames()
			if len(names) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "no profiles (add one with `cc-mux set <name> KEY=VALUE --create`)")
				return nil
			}
			tw := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
			for _, name := range names {
				fmt.Fprintf(tw, "%s\t%s\n", name, cfg.Profiles[name].Description)
			}
			return tw.Flush()
		},
	}
}

func newShowCmd() *cobra.Command {
	var reveal, resolved bool
	cmd := &cobra.Command{
		Use:   "show <profile>",
		Short: "Show a profile's env (secrets masked by default)",
		Long: `Show a profile's command and env entries. By default the configured values
are printed verbatim (${VAR} references are shown as-is and are not secret).

  --resolved   interpolate ${VAR} against the host environment
  --reveal     with --resolved, print real values instead of masking them`,
		Example: "  cc-mux show openrouter\n  cc-mux show openrouter --resolved\n  cc-mux show openrouter --resolved --reveal",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			name := args[0]
			p, ok := cfg.Profiles[name]
			if !ok {
				return fmt.Errorf("unknown profile %q", name)
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "profile: %s\n", name)
			if p.Description != "" {
				fmt.Fprintf(out, "desc:    %s\n", p.Description)
			}
			fmt.Fprintf(out, "command: %s\n", cfg.CommandFor(p))
			if len(p.Args) > 0 {
				fmt.Fprintf(out, "args:    %v\n", p.Args)
			}

			values := p.Env
			if resolved {
				var missing []string
				values, missing = p.ResolveEnv(config.OsLookup)
				if len(missing) > 0 && !reveal {
					fmt.Fprintf(out, "warning: unresolved: %v\n", missing)
				}
			}
			fmt.Fprintln(out, "env:")
			keys := make([]string, 0, len(values))
			for k := range values {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			for _, k := range keys {
				fmt.Fprintf(out, "  %s=%s\n", k, displayValue(values[k], resolved && !reveal))
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&reveal, "reveal", false, "show real values (only meaningful with --resolved)")
	cmd.Flags().BoolVar(&resolved, "resolved", false, "interpolate ${VAR} against the host environment")
	return cmd
}

func newPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "path",
		Short:   "Print the config file path",
		Args:    cobra.NoArgs,
		Example: "  cc-mux path",
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintln(cmd.OutOrStdout(), config.Path())
		},
	}
}

// displayValue masks a resolved value unless it is empty or a ${VAR} reference.
func displayValue(v string, mask bool) string {
	if v == "" {
		return "(empty)"
	}
	if !mask {
		return v
	}
	if len(v) <= 4 {
		return "••••"
	}
	return "••••" + v[len(v)-4:]
}
