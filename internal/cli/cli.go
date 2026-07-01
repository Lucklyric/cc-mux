// Package cli wires the cc-mux command-line interface (built on cobra) to the
// config, launch, doctor, and update packages.
package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

// Build metadata, overridden at release time via -ldflags.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

const rootLong = `cc-mux keeps every Claude Code environment you use — different providers,
proxies, tokens, and flags — as named profiles in one config file, and launches
claude with a profile's environment inlined.

  cc-mux openrouter
  ⇒ ANTHROPIC_BASE_URL="https://openrouter.ai/api" ANTHROPIC_AUTH_TOKEN="…" \
    ANTHROPIC_API_KEY="" CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1 claude

Any first argument that is not a subcommand is treated as a profile name, so
` + "`cc-mux openrouter`" + ` is shorthand for ` + "`cc-mux run openrouter`" + `. Trailing arguments are
passed through to claude verbatim (use -- to separate): ` + "`cc-mux openrouter -- --model x`" + `.

Config lives at ~/.config/cc-mux/config.json (override with $CC_MUX_CONFIG).
Env values support ${VAR} interpolation from the host environment, so secrets
stay in your shell/keychain rather than in the file. Run ` + "`cc-mux doctor`" + ` to
validate, ` + "`cc-mux help <command>`" + ` for detailed, example-driven help.`

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "cc-mux <profile|command> [args]",
		Short:         "Profile-based launcher & config manager for Claude Code",
		Long:          rootLong,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("cc-mux {{.Version}}\n")
	root.AddCommand(
		newRunCmd(),
		newListCmd(),
		newShowCmd(),
		newDoctorCmd(),
		newEditCmd(),
		newSetCmd(),
		newInitCmd(),
		newPathCmd(),
		newUpdateCmd(),
		newVersionCmd(),
	)
	return root
}

// Execute is the CLI entry point.
func Execute() {
	root := newRootCmd()
	args := preprocess(os.Args[1:], knownNames(root))
	root.SetArgs(args)
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "cc-mux: "+err.Error())
		os.Exit(1)
	}
}

// preprocess rewrites `cc-mux <profile> …` into `cc-mux run <profile> …` when
// the first argument is not a known subcommand or a flag.
func preprocess(args, known []string) []string {
	if len(args) == 0 {
		return args
	}
	first := args[0]
	if strings.HasPrefix(first, "-") {
		return args // --help / --version / other root flags
	}
	for _, name := range known {
		if first == name {
			return args
		}
	}
	return append([]string{"run"}, args...)
}

func knownNames(root *cobra.Command) []string {
	var names []string
	for _, c := range root.Commands() {
		names = append(names, c.Name())
		names = append(names, c.Aliases...)
	}
	names = append(names, "help", "completion") // cobra built-ins
	return names
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version, commit, and build date",
		Args:  cobra.NoArgs,
		Run: func(cmd *cobra.Command, _ []string) {
			fmt.Fprintf(cmd.OutOrStdout(), "cc-mux %s (commit %s, built %s)\n", version, commit, date)
		},
	}
}
