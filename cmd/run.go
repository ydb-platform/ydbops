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
	description    *command.Description
	commandOptions *options.RunOptions
	cobraCommand   *cobra.Command
	restarter      *restarters.RunRestarter // TODO(shmel1k@): move to restarter interface.
}

func NewRunCommand(
	description *command.Description,
	restarter *restarters.RunRestarter,
) command.Command {
	return &RunCommand{
		description: description,
		commandOptions: &options.RunOptions{
			RestartOptions: &options.RestartOptions{},
		}, // TODO(shmel1k@): remove from options package.
		restarter: restarters.NewRunRestarter(options.Logger), // TODO(shmel1k@): remove link to global variable
	}
}

func (r *RunCommand) RegisterSubcommands(opts *command.BaseOptions, c ...command.Command) {
	for _, v := range c {
		r.ToCobraCommand(opts).AddCommand(v.ToCobraCommand(opts))
	}
}

func (r *RunCommand) RegisterOptions(opts *command.BaseOptions) {
	r.commandOptions.DefineFlags(r.ToCobraCommand(opts).PersistentFlags())
}

func (r *RunCommand) RunCallback(opts *command.BaseOptions) func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("Free args not expected: %v", args)
		}

		err := client.InitConnectionFactory(
			*opts,
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
	}
}

func (r *RunCommand) ToCobraCommand(opts *command.BaseOptions) *cobra.Command {
	if r.cobraCommand != nil {
		return r.cobraCommand
	}
	r.cobraCommand = &cobra.Command{
		Use:     r.description.GetUse(),
		Short:   r.description.GetShortDescription(),
		Long:    r.description.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(opts, r.restarter.Opts),
		RunE:    r.RunCallback(opts),
	}
	return r.cobraCommand
}
