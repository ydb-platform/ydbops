package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
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
			var storageRestarter restarters.Restarter
			var tenantRestarter restarters.Restarter

			if rootOpts.KubeconfigPath != "" {
				storageRestarter = restarters.NewStorageK8sRestarter(
					options.Logger,
					rootOpts.KubeconfigPath,
					rootOpts.K8sNamespace,
				)
				tenantRestarter = restarters.NewTenantK8sRestarter(
					options.Logger,
					rootOpts.KubeconfigPath,
					rootOpts.K8sNamespace,
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

			var err error

			bothUnspecified := !restartOpts.Storage && !restartOpts.Tenant

			if restartOpts.Storage || bothUnspecified {
				err = rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, storageRestarter)
			}

			if err == nil && (restartOpts.Tenant || bothUnspecified) {
				err = rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, tenantRestarter)
			}

			return err
		},
	})

	restartOpts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
