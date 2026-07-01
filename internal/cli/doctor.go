package cli

import (
	"fmt"
	"os/exec"

	"github.com/Lucklyric/cc-mux/internal/config"
	"github.com/Lucklyric/cc-mux/internal/doctor"
	"github.com/spf13/cobra"
)

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor [profile]",
		Short: "Validate the config and profiles",
		Long: `Check the config for problems: unsupported schema version, unresolved
${VAR} references, launch commands missing from PATH, malformed base URLs, and
profile names that shadow subcommands. Exits non-zero if any error is found.`,
		Example: "  cc-mux doctor\n  cc-mux doctor openrouter",
		Args:    cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			profile := ""
			if len(args) == 1 {
				profile = args[0]
			}
			findings := doctor.Check(cfg, profile, config.OsLookup, exec.LookPath)
			out := cmd.OutOrStdout()
			for _, f := range findings {
				where := ""
				if f.Profile != "" {
					where = "[" + f.Profile + "] "
				}
				fmt.Fprintf(out, "%-5s %s%s\n", f.Level, where, f.Message)
			}
			if doctor.HasErrors(findings) {
				return fmt.Errorf("doctor found problems")
			}
			fmt.Fprintln(out, "ok")
			return nil
		},
	}
}
