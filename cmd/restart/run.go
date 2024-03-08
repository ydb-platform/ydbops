package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/cobra_util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

func NewRunCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewRunRestarter(options.Logger)

	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "run",
		Short: "Run an arbitrary executable (e.g. shell code) in the context of the current host",
		Long: `ydb-ops restart run:
	Run an arbitrary executable (e.g. shell code) in the context of the current host.
	For every host released by CMS, ydb-ops will execute this payload independently.

	Restart will be treated as successful if your executable finished with a zero 
	return code.

	Certain environment variable will be passed to your executable on each run:
		$HOSTNAME: the fqdn of the node currently released by CMS.`,
		Run: func(cmd *cobra.Command, args []string) {
			rolling.PrepareRolling(*restartOpts, *rootOpts, options.Logger, restarter)
		},
	}, restarter.Opts)

	restarter.Opts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
