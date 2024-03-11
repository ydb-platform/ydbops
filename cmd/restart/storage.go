package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/internal/cobra_util"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewStorageCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance

	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "storage",
		Short: "Restarts a specified subset of tenant nodes",
    Long:  `ydbops restart storage:
  Restarts a specified subset of storage nodes`,
	}, restartOpts)

	return cmd
}

func init() {
}
