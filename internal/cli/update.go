package cli

import (
	"github.com/Lucklyric/cc-mux/internal/update"
	"github.com/spf13/cobra"
)

func newUpdateCmd() *cobra.Command {
	var check bool
	cmd := &cobra.Command{
		Use:   "update",
		Short: "Self-update from GitHub releases",
		Long: `Download the latest release for this OS/arch, verify its SHA-256 against
the release checksums, and atomically replace the running binary.

The source repo defaults to Lucklyric/cc-mux and can be overridden with the
CC_MUX_UPDATE_REPO environment variable (owner/name).`,
		Example: "  cc-mux update\n  cc-mux update --check\n  CC_MUX_UPDATE_REPO=you/fork cc-mux update",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return update.Run(version, check, cmd.OutOrStdout())
		},
	}
	cmd.Flags().BoolVar(&check, "check", false, "report the latest version without installing")
	return cmd
}
