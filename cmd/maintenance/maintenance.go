package maintenance

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
)

type MaintenanceOptions struct {
	*command.BaseOptions
}

func NewMaintenanceCommand() *cobra.Command {
	options := &MaintenanceOptions{}
	cmd := &cobra.Command{
		Use:   "maintenance",
		Short: "Request hosts from the Cluster Management System",
		Long: `ydbops maintenance [command]:
    Manage host maintenance operations: request and return hosts
    with performed maintenance back to the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			options.BaseOptions, options,
		),
		RunE: cli.RequireSubcommand,
	}

	cmd = cli.SetDefaultsOn(&cobra.Command{})
	return cmd
}
