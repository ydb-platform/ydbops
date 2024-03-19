package restart

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
)

func NewStorageCmd() *cobra.Command {
	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "storage",
		Short: "Restarts a specified subset of storage nodes",
		Long: `ydbops restart storage:
  Restarts a specified subset of storage nodes.
  Not specifying any filters will restart all storage nodes.`,
		RunE: cli.RequireSubcommand,
	})

	return cmd
}

func init() {
}
