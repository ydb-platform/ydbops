package restart

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

var RestartCommandDescription = command.NewDescription(
	"restart",
	"Restarts a specified subset of nodes in the cluster",
	`ydbops restart:
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to
  narrow down what subset will be restarted.`)

func New(
	description *command.Description,
	f cmdutil.Factory,
) *cobra.Command {
	opts := &RestartOptions{
		BaseOptions: &command.BaseOptions{},
	}
	cmd := &cobra.Command{
		Use:     description.GetUse(),
		Short:   description.GetShortDescription(),
		Long:    description.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(opts.BaseOptions, opts.RollingRestartOptions),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			var storageRestarter restarters.Restarter
			var tenantRestarter restarters.Restarter

			if opts.RollingRestartOptions.KubeconfigPath != "" {
				storageRestarter = restarters.NewStorageK8sRestarter(
					options.Logger,
					opts.RollingRestartOptions.KubeconfigPath,
					opts.RollingRestartOptions.K8sNamespace,
				)
				tenantRestarter = restarters.NewTenantK8sRestarter(
					options.Logger,
					opts.RollingRestartOptions.KubeconfigPath,
					opts.RollingRestartOptions.K8sNamespace,
				)
			} else {
				storageRestarter = restarters.NewStorageSSHRestarter(
					options.Logger,
					opts.RollingRestartOptions.SSHArgs,
					opts.RollingRestartOptions.CustomSystemdUnitName,
				)
				tenantRestarter = restarters.NewTenantSSHRestarter(
					options.Logger,
					opts.RollingRestartOptions.SSHArgs,
					opts.RollingRestartOptions.CustomSystemdUnitName,
				)
			}

			bothUnspecified := !opts.RollingRestartOptions.Storage && !opts.RollingRestartOptions.Tenant

			var executer rolling.Executer
			var err error
			if opts.RollingRestartOptions.Storage || bothUnspecified {
				// TODO(shmel1k@): add logger to NewExecuter parameters
				executer = rolling.NewExecuter(opts.RollingRestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), storageRestarter)
				err = executer.Execute()
			}

			if err == nil && (opts.RollingRestartOptions.Tenant || bothUnspecified) {
				executer = rolling.NewExecuter(opts.RollingRestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), tenantRestarter)
			}
			err = executer.Execute()
			return err
		},
	}

	return cmd
}
