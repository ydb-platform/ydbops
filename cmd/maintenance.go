package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewMaintenanceCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance
	restartOpts := options.RestartOptionsInstance

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "maintenance",
		Short: "Request hosts from the Cluster Management System",
		Long: `ydbops maintenance [command]: 
    Manage host maintenance operations: request and return hosts
    with performed maintenance back to the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			restartOpts, rootOpts,
		),
		RunE: cli.RequireSubcommand,
	})

	restartOpts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
