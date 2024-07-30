package restart

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

var RestartCommandDescription = command.NewDescription(
	"restart",
	"Restarts a specified subset of nodes in the cluster",
	`ydbops restart:
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to
  narrow down what subset will be restarted.`)

func New(
	f cmdutil.Factory,
) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}
	cmd := &cobra.Command{
		Use:     RestartCommandDescription.GetUse(),
		Short:   RestartCommandDescription.GetShortDescription(),
		Long:    RestartCommandDescription.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(f.GetBaseOptions(), opts.RestartOptions),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			err := opts.Validate()
			if err != nil {
				return err
			}

			var storageRestarter restarters.Restarter
			var tenantRestarter restarters.Restarter

			if opts.RestartOptions.KubeconfigPath != "" {
				storageRestarter = restarters.NewStorageK8sRestarter(
					options.Logger,
					&restarters.StorageK8sRestarterOptions{
						K8sRestarterOptions: &restarters.K8sRestarterOptions{
							KubeconfigPath:  opts.RestartOptions.KubeconfigPath,
							Namespace:       opts.RestartOptions.K8sNamespace,
							RestartDuration: time.Duration(opts.RestartOptions.RestartDuration) * time.Second,
						},
					},
				)
				tenantRestarter = restarters.NewTenantK8sRestarter(
					options.Logger,
					&restarters.TenantK8sRestarterOptions{
						K8sRestarterOptions: &restarters.K8sRestarterOptions{
							KubeconfigPath:  opts.RestartOptions.KubeconfigPath,
							Namespace:       opts.RestartOptions.K8sNamespace,
							RestartDuration: time.Duration(opts.RestartOptions.RestartDuration) * time.Second,
						},
					},
				)
			} else {
				storageRestarter = restarters.NewStorageSSHRestarter(
					options.Logger,
					opts.RestartOptions.SSHArgs,
					opts.RestartOptions.CustomSystemdUnitName,
				)
				tenantRestarter = restarters.NewTenantSSHRestarter(
					options.Logger,
					opts.RestartOptions.SSHArgs,
					opts.RestartOptions.CustomSystemdUnitName,
				)
			}

			bothUnspecified := !opts.RestartOptions.Storage && !opts.RestartOptions.Tenant

			var executer rolling.Executer
			if opts.RestartOptions.Storage || bothUnspecified {
				// TODO(shmel1k@): add logger to NewExecuter parameters
				executer = rolling.NewExecuter(opts.RestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), storageRestarter)
				err = executer.Execute()
			}

			if err != nil {
				return err
			}

			if opts.RestartOptions.Tenant || bothUnspecified {
				executer = rolling.NewExecuter(opts.RestartOptions, zap.S(), f.GetCMSClient(), f.GetDiscoveryClient(), tenantRestarter)
				err = executer.Execute()
			}

			return err
		},
	}
	opts.DefineFlags(cmd.Flags())

	return cmd
}
