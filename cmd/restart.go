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

var RestartCommandDescription = command.NewDescription(
	"restart",
	"Restarts a specified subset of nodes in the cluster",
	`ydbops restart:
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to
  narrow down what subset will be restarted.`)

type RestartCommand struct {
	description    *command.BaseCommandDescription
	preRunCallback cli.PreRunCallback
	rootOptions    *options.RootOptions
	commandOptions *options.RestartOptions
	cobraCommand   *cobra.Command
}

func NewRestartCommand(
	description *command.BaseCommandDescription,
	rootOptions *options.RootOptions,
	commandOptions *options.RestartOptions,
	preRunCallback cli.PreRunCallback,
) command.Command {
	return &RestartCommand{
		description:    description,
		rootOptions:    rootOptions,
		commandOptions: commandOptions,
		preRunCallback: preRunCallback,
	}
}

func (r *RestartCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

func (r *RestartCommand) run(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return fmt.Errorf("Free args not expected: %v", args)
	}

	var storageRestarter restarters.Restarter
	var tenantRestarter restarters.Restarter

	err := client.InitConnectionFactory(
		*r.rootOptions,
		options.Logger,
		options.DefaultRetryCount,
	)
	if err != nil {
		return err
	}

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

	if r.commandOptions.Storage || bothUnspecified {
		err = rolling.ExecuteRolling(*r.commandOptions, options.Logger, storageRestarter)
	}

	if err == nil && (r.commandOptions.Tenant || bothUnspecified) {
		err = rolling.ExecuteRolling(*r.commandOptions, options.Logger, tenantRestarter)
	}

	return err
}

func (r *RestartCommand) ToCobraCommand() *cobra.Command {
	if r.cobraCommand == nil {
		r.cobraCommand = &cobra.Command{
			Use:     r.description.GetUse(),
			Short:   r.description.GetShortDescription(),
			Long:    r.description.GetLongDescription(),
			PreRunE: r.preRunCallback(r.rootOptions, r.commandOptions),
			RunE:    r.run,
		}
	}
	return r.cobraCommand
}

func NewRestartCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance
	restartOpts := options.RestartOptionsInstance

	restartCommand := NewRestartCommand(
		RestartCommandDescription,
		rootOpts,
		restartOpts,
		cli.PopulateProfileDefaultsAndValidate,
	)

	cmd := cli.SetDefaultsOn(restartCommand.ToCobraCommand())

	restartOpts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
