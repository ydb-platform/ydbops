package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type RunCommand struct {
	description    *command.BaseCommandDescription
	rootOptions    *options.RootOptions
	commandOptions *options.RunOptions
	cobraCommand   *cobra.Command
	restarter      *restarters.RunRestarter // TODO(shmel1k@): move to restarter interface.
}

func NewRunCommand(
	description *command.BaseCommandDescription,
	rootOptions *options.RootOptions, // TODO(shmel1k@): embed commandOptions from rootOptions
	commandOptions *options.RunOptions,
	restarter *restarters.RunRestarter,
) command.Command {
	return &RunCommand{
		description:    description,
		rootOptions:    rootOptions,
		commandOptions: commandOptions,
	}
}

func (r *RunCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

func (r *RunCommand) ToCobraCommand() *cobra.Command {
	if r.cobraCommand != nil {
		return r.cobraCommand
	}
	r.cobraCommand = &cobra.Command{
		Use:     r.description.GetUse(),
		Short:   r.description.GetShortDescription(),
		Long:    r.description.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(r.commandOptions.RestartOptions, r.rootOptions, r.restarter.Opts),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			err := client.InitConnectionFactory(
				*r.rootOptions,
				options.Logger,
				options.DefaultRetryCount,
			)
			if err != nil {
				return err
			}

			bothUnspecified := !r.commandOptions.Storage && !r.commandOptions.Tenant

			if r.commandOptions.Storage || bothUnspecified {
				r.restarter.SetStorageOnly()
				err = rolling.ExecuteRolling(*r.commandOptions.RestartOptions, options.Logger, r.restarter)
			}

			if err == nil && (r.commandOptions.Tenant || bothUnspecified) {
				r.restarter.SetDynnodeOnly()
				err = rolling.ExecuteRolling(*r.commandOptions.RestartOptions, options.Logger, r.restarter)
			}

			return err
		},
	}
	return r.cobraCommand
}

func NewRunCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	restarter := restarters.NewRunRestarter(options.Logger)
	runCommand := NewRunCommand(
		command.NewDescription(
			"run",
			"Run an arbitrary executable (e.g. shell code) in the context of the local machine",
			`ydbops restart run:
	Run an arbitrary executable (e.g. shell code) in the context of the local machine
	(where rolling-restart is launched). For example, if you want to execute ssh commands
	on the ydb cluster node, you must write ssh commands yourself. See the examples.

	For every node released by CMS, ydbops will execute this payload independently.

	Restart will be treated as successful if your executable finished with a zero
	return code.

	Certain environment variable will be passed to your executable on each run:
		$HOSTNAME: the fqdn of the node currently released by CMS.`,
		),
		options.RootOptionsInstance,
		&options.RunOptions{
			RestartOptions: restartOpts,
		},
		restarter,
	)

	cmd := cli.SetDefaultsOn(runCommand.ToCobraCommand())

	restarter.Opts.DefineFlags(cmd.Flags())
	restartOpts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
