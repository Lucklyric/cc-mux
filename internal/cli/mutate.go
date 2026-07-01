package cli

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/Lucklyric/cc-mux/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd() *cobra.Command {
	var force bool
	cmd := &cobra.Command{
		Use:     "init",
		Short:   "Write a starter config with a documented example profile",
		Args:    cobra.NoArgs,
		Example: "  cc-mux init\n  cc-mux init --force",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.Path()
			if _, err := os.Stat(path); err == nil && !force {
				return fmt.Errorf("config already exists at %s (use --force to overwrite)", path)
			}
			if err := config.Default().SaveTo(path); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "wrote starter config to %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVar(&force, "force", false, "overwrite an existing config")
	return cmd
}

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit",
		Short: "Open the config in $EDITOR, then validate it",
		Long: `Open the config file in $VISUAL/$EDITOR (falling back to vi). Creates a
starter config first if none exists. After the editor exits, the file is parsed
so syntax errors are reported immediately.`,
		Args:    cobra.NoArgs,
		Example: "  cc-mux edit\n  EDITOR=nano cc-mux edit",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path := config.Path()
			if _, err := os.Stat(path); os.IsNotExist(err) {
				if err := config.Default().SaveTo(path); err != nil {
					return err
				}
				fmt.Fprintf(cmd.OutOrStdout(), "created %s\n", path)
			}
			editor := firstNonEmpty(os.Getenv("VISUAL"), os.Getenv("EDITOR"), "vi")
			ed := exec.Command(editor, path)
			ed.Stdin, ed.Stdout, ed.Stderr = os.Stdin, os.Stdout, os.Stderr
			if err := ed.Run(); err != nil {
				return fmt.Errorf("editor %q: %w", editor, err)
			}
			if _, err := config.LoadFrom(path); err != nil {
				return fmt.Errorf("config is invalid after edit: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), "config is valid")
			return nil
		},
	}
}

func newSetCmd() *cobra.Command {
	var create bool
	cmd := &cobra.Command{
		Use:   "set <profile> KEY=VALUE [KEY=VALUE ...]",
		Short: "Add or update one or more env entries on a profile",
		Long: `Set one or more environment entries on a profile in a single call. Values
may contain ${VAR} references, which are stored verbatim and resolved at launch.
Use --create to add a new profile if it does not exist yet.`,
		Example: "  cc-mux set openrouter ANTHROPIC_BASE_URL=http://host:8788/\n" +
			"  cc-mux set openrouter ANTHROPIC_AUTH_TOKEN='${OPENROUTER_API_KEY}' ANTHROPIC_API_KEY=\n" +
			"  cc-mux set newprofile ANTHROPIC_BASE_URL=http://x --create",
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			name, pairs := args[0], args[1:]
			cfg, err := config.Load()
			if err != nil {
				if create {
					cfg = config.Default()
					cfg.Profiles = map[string]*config.Profile{}
				} else {
					return err
				}
			}
			p, ok := cfg.Profiles[name]
			if !ok {
				if !create {
					return fmt.Errorf("unknown profile %q (use --create to add it)", name)
				}
				p = &config.Profile{Env: map[string]string{}}
				cfg.Profiles[name] = p
			}
			if p.Env == nil {
				p.Env = map[string]string{}
			}
			for _, pair := range pairs {
				key, val, ok := strings.Cut(pair, "=")
				if !ok || key == "" {
					return fmt.Errorf("invalid KEY=VALUE pair: %q", pair)
				}
				p.Env[key] = val
			}
			if err := cfg.Save(); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "updated profile %q (%d entries set)\n", name, len(pairs))
			return nil
		},
	}
	cmd.Flags().BoolVar(&create, "create", false, "create the profile (and config) if missing")
	return cmd
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
