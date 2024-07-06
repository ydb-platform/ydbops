package run

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

var RunCommandDescription = command.NewDescription(
	"run",
	"Run an arbitrary executable (e.g. shell code) in the context of the local machine",
	`ydbops restart run:
		Run an arbitrary executable (e.g. shell code) in the context of the local machine
		(where rolling-restart is launched). For example, if you want to execute ssh commands
		on the ydb cluster node, you must write ssh commands yourself. See the examples.

		For every node released by CMS, ydbops will execute this payload independently.

		Restart will be treated as successful if your executable finished with a zero
		return code.

		Certain environment variable will be passed to your executable on each run:
			$HOSTNAME: the fqdn of the node currently released by CMS.`,
)

func New(
	f cmdutil.Factory,
) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}
	cmd := &cobra.Command{
		Use:     RunCommandDescription.GetUse(),
		Short:   RunCommandDescription.GetShortDescription(),
		Long:    RunCommandDescription.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(f.GetBaseOptions(), opts),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			bothUnspecified := !opts.RestartOptions.Storage && !opts.RestartOptions.Tenant

			restarter := restarters.NewRunRestarter(zap.S(), &restarters.RunRestarterParams{
				PayloadFilePath: opts.PayloadFilePath,
			})

			var executer rolling.Executer
			var err error
			if opts.RestartOptions.Storage || bothUnspecified {
				restarter.SetStorageOnly()
				executer = rolling.NewExecuter(opts.RestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
				err = executer.Execute()
			}

			if err == nil && (opts.RestartOptions.Tenant || bothUnspecified) {
				restarter.SetDynnodeOnly()
				executer = rolling.NewExecuter(opts.RestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
				err = executer.Execute()
			}

			return err
		},
	}

	opts.DefineFlags(cmd.Flags())
	return cmd
}
