package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
)

var RestartCommandDescription = command.NewDescription(
	"restart",
	"Restarts a specified subset of nodes in the cluster",
	`ydbops restart:
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to
  narrow down what subset will be restarted.`)

type restartCommandOptions struct {
	// TODO(shmel1k@): remove from options package.
	*options.RestartOptions
}

type RestartCommand struct {
	*command.Base
	description    *command.Description
	preRunCallback cli.PreRunCallback
	commandOptions *options.RestartOptions
	cobraCommand   *cobra.Command
	f              cmdutil.Factory
}

func NewRestartCommand(
	description *command.Description,
	rootCommand *command.Base,
	f cmdutil.Factory,
) command.Command {
	return &RestartCommand{
		description:    description,
		preRunCallback: cli.PopulateProfileDefaultsAndValidate,
		commandOptions: &options.RestartOptions{},
		Base:           rootCommand,
	}
}

func (r *RestartCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		// TODO(shmel1k@): remove copypaste
		cli.SetDefaultsOn(v.ToCobraCommand())
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

func (r *RestartCommand) RegisterOptions() {
	// TODO(shmel1k@): less letters.
	r.commandOptions.DefineFlags(r.ToCobraCommand().PersistentFlags())
}

func (r *RestartCommand) RunCallback() func(_ *cobra.Command, _ []string) error {
	return func(_ *cobra.Command, args []string) error {
		if len(args) > 0 {
			return fmt.Errorf("Free args not expected: %v", args)
		}

		var storageRestarter restarters.Restarter
		var tenantRestarter restarters.Restarter

		if r.commandOptions.KubeconfigPath != "" {
			storageRestarter = restarters.NewStorageK8sRestarter(
				options.Logger,
				r.commandOptions.KubeconfigPath,
				r.commandOptions.K8sNamespace,
			)
			tenantRestarter = restarters.NewTenantK8sRestarter(
				options.Logger,
				r.commandOptions.KubeconfigPath,
				r.commandOptions.K8sNamespace,
			)
		} else {
			storageRestarter = restarters.NewStorageSSHRestarter(
				options.Logger,
				r.commandOptions.SSHArgs,
				r.commandOptions.CustomSystemdUnitName,
			)
			tenantRestarter = restarters.NewTenantSSHRestarter(
				options.Logger,
				r.commandOptions.SSHArgs,
				r.commandOptions.CustomSystemdUnitName,
			)
		}

		bothUnspecified := !r.commandOptions.Storage && !r.commandOptions.Tenant

		var executer rolling.Executer
		var err error
		if r.commandOptions.Storage || bothUnspecified {
			// TODO(shmel1k@): add logger to NewExecuter parameters
			executer = rolling.NewExecuter(*r.commandOptions, zap.S(), r.f.GetCMSClient(), r.f.GetDiscoveryClient(), storageRestarter)
		}

		if err == nil && (r.commandOptions.Tenant || bothUnspecified) {
			executer = rolling.NewExecuter(*r.commandOptions, zap.S(), r.f.GetCMSClient(), r.f.GetDiscoveryClient(), tenantRestarter)
		}
		err = executer.Execute()

		return err
	}
}

func (r *RestartCommand) ToCobraCommand() *cobra.Command {
	if r.cobraCommand == nil {
		r.cobraCommand = &cobra.Command{
			Use:     r.description.GetUse(),
			Short:   r.description.GetShortDescription(),
			Long:    r.description.GetLongDescription(),
			PreRunE: r.preRunCallback(r.Base.GetBaseOptions()),
			RunE:    r.RunCallback(),
		}
	}
	return r.cobraCommand
}
