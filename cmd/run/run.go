package run

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

func NewRunCommand(
	description *command.Description,
	f cmdutil.Factory,
) *cobra.Command {
	opts := &RunOptions{
		BaseOptions:           &command.BaseOptions{},
		RollingRestartOptions: &rolling.RollingRestartOptions{},
	}
	cmd := &cobra.Command{
		Use:     description.GetUse(),
		Short:   description.GetShortDescription(),
		Long:    description.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(opts.BaseOptions, opts),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			bothUnspecified := !opts.RollingRestartOptions.Storage && !opts.RollingRestartOptions.Tenant

			restarter := restarters.NewRunRestarter(zap.S(), &restarters.RunRestarterParams{
				PayloadFilePath: opts.PayloadFilePath,
			})

			var executer rolling.Executer
			var err error
			if opts.RollingRestartOptions.Storage || bothUnspecified {
				restarter.SetStorageOnly()
				executer = rolling.NewExecuter(opts.RollingRestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
				err = executer.Execute()
			}

			if err == nil && (opts.RollingRestartOptions.Tenant || bothUnspecified) {
				restarter.SetDynnodeOnly()
				executer = rolling.NewExecuter(opts.RollingRestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
				err = executer.Execute()
			}

			return err
		},
	}
	return cmd
}
