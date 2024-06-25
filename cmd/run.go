package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

type RunCommand struct {
	*command.Base
	description    *command.Description
	commandOptions *options.RunOptions
	cobraCommand   *cobra.Command
	restarter      *restarters.RunRestarter // TODO(shmel1k@): move to restarter interface.
	f              cmdutil.Factory
}

func NewRunCommand(
	description *command.Description,
	rootCommand *command.Base,
	restarter *restarters.RunRestarter,
	f cmdutil.Factory,
) command.Command {
	return &RunCommand{
		description: description,
		commandOptions: &options.RunOptions{
			RestartOptions: &options.RestartOptions{},
		}, // TODO(shmel1k@): remove from options package.
		restarter: restarters.NewRunRestarter(options.Logger), // TODO(shmel1k@): remove link to global variable
		Base:      rootCommand,
	}
}

func (r *RunCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

func (r *RunCommand) RegisterOptions() {
	r.commandOptions.DefineFlags(r.ToCobraCommand().PersistentFlags())
}

func (r *RunCommand) RunCallback() func(*cobra.Command, []string) error {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("Free args not expected: %v", args)
		}

		bothUnspecified := !r.commandOptions.Storage && !r.commandOptions.Tenant

		var executer rolling.Executer
		var err error
		if r.commandOptions.Storage || bothUnspecified {
			r.restarter.SetStorageOnly()
			executer = rolling.NewExecuter(*r.commandOptions.RestartOptions, options.Logger, r.f.GetCMSClient(), r.f.GetDiscoveryClient(), r.restarter)
			err = executer.Execute()
		}

		if err == nil && (r.commandOptions.Tenant || bothUnspecified) {
			r.restarter.SetDynnodeOnly()
			executer = rolling.NewExecuter(*r.commandOptions.RestartOptions, options.Logger, r.f.GetCMSClient(), r.f.GetDiscoveryClient(), r.restarter)
			err = executer.Execute()
		}

		return err
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
		PreRunE: cli.PopulateProfileDefaultsAndValidate(r.GetBaseOptions(), r.restarter.Opts),
		RunE:    r.RunCallback(),
	}
	return r.cobraCommand
}
