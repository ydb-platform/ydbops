package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

func NewRunCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance
	restartOpts := options.RestartOptionsInstance
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
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			restartOpts, rootOpts, restarter.Opts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			err := client.InitConnectionFactory(
				*rootOpts,
				options.Logger,
				options.DefaultRetryCount,
			)
			if err != nil {
				return err
			}

			bothUnspecified := !restartOpts.Storage && !restartOpts.Tenant

			if restartOpts.Storage || bothUnspecified {
				restarter.SetStorageOnly()
				err = rolling.ExecuteRolling(*restartOpts, options.Logger, restarter)
			}

			if err == nil && (restartOpts.Tenant || bothUnspecified) {
				restarter.SetDynnodeOnly()
				err = rolling.ExecuteRolling(*restartOpts, options.Logger, restarter)
			}

			return err
		},
	})

	restarter.Opts.DefineFlags(cmd.Flags())
	restartOpts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
