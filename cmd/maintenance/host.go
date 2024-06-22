package maintenance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewHostCommand(
	description *command.Description,
	rootCommand *command.Base,
	preRunCallback cli.PreRunCallback,
) command.Command {
	return &HostCommand{
		Base:           rootCommand,
		preRunCallback: preRunCallback,
		commandOptions: &options.MaintenanceHostOpts{},
	}
}

type HostCommand struct {
	*command.Base
	description    *command.Description
	preRunCallback cli.PreRunCallback
	commandOptions *options.MaintenanceHostOpts
	cobraCommand   *cobra.Command
}

func (h *HostCommand) ToCobraCommand() *cobra.Command {
	if h.cobraCommand != nil {
		return h.cobraCommand
	}

	h.cobraCommand = &cobra.Command{
		Use:   "host",
		Short: "Request host from the CMS (Cluster Management System)",
		Long: `ydbops maintenance host:
  Make a request to take the host out of the cluster.`,
		RunE: h.RunCallback(),
	}

	return h.cobraCommand
}

func (h *HostCommand) RegisterOptions() {
	h.commandOptions.DefineFlags(h.ToCobraCommand().PersistentFlags())
}

func (h *HostCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		cli.SetDefaultsOn(v.ToCobraCommand())
		h.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

func (h *HostCommand) RunCallback() func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, _ []string) error {
		err := client.InitConnectionFactory(
			*h.GetBaseOptions(),
			options.Logger,
			options.DefaultRetryCount,
		)
		if err != nil {
			return err
		}

		taskId, err := maintenance.RequestHost(h.commandOptions)
		if err != nil {
			return err
		}

		fmt.Printf(
			"Your task id is:\n\n%s\n\nPlease write it down for refreshing and completing the task later.\n",
			taskId,
		)

		return nil

	}
}
