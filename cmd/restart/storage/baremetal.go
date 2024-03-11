package storage

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/internal/cobra_util"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

func NewStorageBaremetalCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewStorageBaremetalRestarter(options.Logger)

	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "baremetal",
		Short: "Restarts a specified subset of storage nodes over SSH",
		Long:  `ydbops restart storage barematal:
  Restarts a specified subset of storage nodes over SSH`,
		Run: func(cmd *cobra.Command, args []string) {
			rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, restarter)
		},
	}, restarter.Opts)

	restarter.Opts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
