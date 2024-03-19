package restart

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

func NewRunCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewRunRestarter(options.Logger)

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "run",
		Short: "Run an arbitrary executable (e.g. shell code) in the context of the local machine",
		Long: `ydbops restart run:
	Run an arbitrary executable (e.g. shell code) in the context of the local machine 
	(where rolling-restart is launched). For example, if you want to execute ssh commands 
	on the ydb cluster node, you must write ssh commands yourself. See the examples.

	For every node released by CMS, ydbops will execute this payload independently.

	Restart will be treated as successful if your executable finished with a zero 
	return code.

	Certain environment variable will be passed to your executable on each run:
		$HOSTNAME: the fqdn of the node currently released by CMS.`,
		PreRunE: cli.ValidateOptions(restartOpts, rootOpts, restarter.Opts),
		Run: func(cmd *cobra.Command, args []string) {
			rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, restarter)
		},
	})

	restarter.Opts.DefineFlags(cmd.Flags())
	restartOpts.TenantOptions.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
