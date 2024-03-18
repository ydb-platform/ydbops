package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/internal/cobra_util"
)

func NewStorageCmd() *cobra.Command {
	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "storage",
		Short: "Restarts a specified subset of storage nodes",
		Long: `ydbops restart storage:
  Restarts a specified subset of storage nodes.
  Not specifying any filters will restart all storage nodes.`,
		RunE: cobra_util.RequireSubcommand,
	})

	return cmd
}

func init() {
}
