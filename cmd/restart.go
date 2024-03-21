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
  Restarts a specified subset of nodes in the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			restartOpts, rootOpts,
		),
		Run: func(cmd *cobra.Command, args []string) {
			var restarter restarters.Restarter

			if rootOpts.KubeconfigPath != "" {
				if restartOpts.Storage {
					restarter = restarters.NewStorageK8sRestarter(
						options.Logger,
						rootOpts.KubeconfigPath,
						rootOpts.K8sNamespace,
					)
				} else { // Tenant
					restarter = restarters.NewTenantK8sRestarter(
						options.Logger,
						rootOpts.KubeconfigPath,
						rootOpts.K8sNamespace,
					)
				}
			} else {
				if restartOpts.Storage {
					restarter = restarters.NewStorageSSHRestarter(
						options.Logger,
						restartOpts.SSHArgs,
						restartOpts.CustomSystemdUnitName,
					)
				} else { // Tenant
					restarter = restarters.NewTenantSSHRestarter(
						options.Logger,
						restartOpts.SSHArgs,
						restartOpts.CustomSystemdUnitName,
					)
				}
			}

			rolling.ExecuteRolling(*restartOpts, *rootOpts, options.Logger, restarter)
		},
	})

	restartOpts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
