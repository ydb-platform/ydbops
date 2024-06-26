package cmd

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

type MaintenanceCommand struct {
	*command.Base
	commandOptions *options.RestartOptions
	cobraCommand   *cobra.Command
}

func (r *MaintenanceCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		v.RegisterOptions()
		cli.SetDefaultsOn(v.ToCobraCommand())
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
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
		RunE: r.RunCallback(),
	})
	r.cobraCommand = cmd
	return r.cobraCommand
}

func (r *MaintenanceCommand) RunCallback() func(*cobra.Command, []string) error {
	return cli.RequireSubcommand
}

func NewMaintenanceCommand(
	base *command.Base,
) command.Command {
	return &MaintenanceCommand{
		commandOptions: &options.RestartOptions{},
		Base:           base,
	}
}
