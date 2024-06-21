package cmd

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

type MaintenanceCommand struct {
	*command.Base
	description    *command.Description
	preRunCallback cli.PreRunCallback
	commandOptions *options.RestartOptions
	cobraCommand   *cobra.Command
}

func (r *MaintenanceCommand) RegisterSubcommands(c ...command.Command) {
}

func (r *MaintenanceCommand) RegisterOptions() {
	r.commandOptions.DefineFlags(r.ToCobraCommand().PersistentFlags())
}

func (r *MaintenanceCommand) ToCobraCommand() *cobra.Command {
	if r.cobraCommand != nil {
		return r.cobraCommand
	}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "maintenance",
		Short: "Request hosts from the Cluster Management System",
		Long: `ydbops maintenance [command]:
    Manage host maintenance operations: request and return hosts
    with performed maintenance back to the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			r.Base.GetBaseOptions(), r.commandOptions,
		),
		RunE: cli.RequireSubcommand,
	})
	return cmd
}

func (r *MaintenanceCommand) RunCallback() func(*cobra.Command, []string) error {
	return cli.RequireSubcommand
}

func NewMaintenanceCommand(
	description *command.Description,
	base *command.Base,
	preRunCallback cli.PreRunCallback, // TODO(shmel1k@): change to validation callback name or smth
) command.Command {
	return &MaintenanceCommand{
		description:    description,
		preRunCallback: preRunCallback,
		commandOptions: &options.RestartOptions{},
		Base:           base,
	}
}
