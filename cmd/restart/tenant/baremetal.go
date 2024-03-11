package tenant

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/cobra_util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

func NewTenantBaremetalCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewTenantBaremetalRestarter(options.Logger)

	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "baremetal",
		Short: "Restarts a specified subset of storage nodes over SSH",
		Long: `ydb-ops restart storage barematal:
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
