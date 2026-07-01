package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/Lucklyric/cc-mux/internal/config"
	"github.com/Lucklyric/cc-mux/internal/launch"
	"github.com/spf13/cobra"
)

func newRunCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "run <profile> [-- args...]",
		Short: "Launch claude with a profile's environment (default action: `cc-mux <profile>`)",
		Long: `Launch the profile's command (claude by default) with the profile's env
inlined on top of your current environment. This is the default action, so
` + "`cc-mux openrouter`" + ` and ` + "`cc-mux run openrouter`" + ` are equivalent.

Arguments after the profile name are passed through to the command verbatim.
Use -- to stop cc-mux from interpreting them. Unresolved ${VAR} references abort
the launch so you never start claude with an empty token.`,
		Example:            "  cc-mux openrouter\n  cc-mux run openrouter\n  cc-mux openrouter -- --model claude-sonnet-4\n  cc-mux openrouter /path/to/project",
		Args:               cobra.MinimumNArgs(1),
		DisableFlagParsing: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if args[0] == "-h" || args[0] == "--help" {
				return cmd.Help()
			}
			name := args[0]
			passthrough := args[1:]
			if len(passthrough) > 0 && passthrough[0] == "--" {
				passthrough = passthrough[1:]
			}
			return runProfile(name, passthrough)
		},
	}
}

func runProfile(name string, passthrough []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("unknown profile %q (run `cc-mux list` to see profiles)", name)
	}

	resolved, missing := p.ResolveEnv(config.OsLookup)
	if len(missing) > 0 {
		return fmt.Errorf("profile %q has unset ${VAR} references: %s", name, strings.Join(missing, ", "))
	}

	command := cfg.CommandFor(p)
	environ := launch.BuildEnviron(os.Environ(), resolved)
	argv := launch.BuildArgv(command, p.Args, passthrough)
	// On success this replaces the process and never returns.
	return launch.Exec(command, argv, environ)
}
