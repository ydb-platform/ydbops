package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

func NewRestartCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance
	restartOpts := options.RestartOptionsInstance

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "restart",
		Short: "Restarts a specified subset of nodes in the cluster",
		Long: `ydbops restart: 
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to 
  narrow down what subset will be restarted.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			restartOpts, rootOpts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			var storageRestarter restarters.Restarter
			var tenantRestarter restarters.Restarter

			err := client.InitConnectionFactory(
				*rootOpts,
				options.Logger,
				options.DefaultRetryCount,
			)
			if err != nil {
				return err
			}

			if restartOpts.KubeconfigPath != "" {
				storageRestarter = restarters.NewStorageK8sRestarter(
					options.Logger,
					restartOpts.KubeconfigPath,
					restartOpts.K8sNamespace,
				)
				tenantRestarter = restarters.NewTenantK8sRestarter(
					options.Logger,
					restartOpts.KubeconfigPath,
					restartOpts.K8sNamespace,
				)
			} else {
				storageRestarter = restarters.NewStorageSSHRestarter(
					options.Logger,
					restartOpts.SSHArgs,
					restartOpts.CustomSystemdUnitName,
				)
				tenantRestarter = restarters.NewTenantSSHRestarter(
					options.Logger,
					restartOpts.SSHArgs,
					restartOpts.CustomSystemdUnitName,
				)
			}

			bothUnspecified := !restartOpts.Storage && !restartOpts.Tenant

			if restartOpts.Storage || bothUnspecified {
				err = rolling.ExecuteRolling(*restartOpts, options.Logger, storageRestarter)
			}

			if err == nil && (restartOpts.Tenant || bothUnspecified) {
				err = rolling.ExecuteRolling(*restartOpts, options.Logger, tenantRestarter)
			}

			return err
		},
	})

	restartOpts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
