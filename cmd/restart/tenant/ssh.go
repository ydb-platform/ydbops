package tenant

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

func NewTenantSSHCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance

	restarter := restarters.NewTenantSSHRestarter(options.Logger)

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "ssh",
		Short: "Restarts a specified subset of tenant nodes over SSH",
		Long: `ydbops restart tenant ssh:
  Restarts a specified subset of tenant nodes over SSH.
  Not specifying any filters will restart all tenant nodes.`,
		PreRunE: cli.ValidateOptions(restartOpts, rootOpts, restarter.Opts),
		Run: func(cmd *cobra.Command, args []string) {
			rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, restarter)
		},
	})

	restarter.Opts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
