package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ydb-platform/ydbops/cmd/maintenance"
	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func addAndReturnCmd(cmd *cobra.Command, rest ...*cobra.Command) *cobra.Command {
	for _, subCmd := range rest {
		cmd.AddCommand(subCmd)
	}
	return cmd
}

func registerAllSubcommands(root *cobra.Command) {
	_ = addAndReturnCmd(root,
		NewRestartCmd(),
		NewRunCmd(),
		addAndReturnCmd(NewMaintenanceCmd(),
			maintenance.NewHostCmd(),
		),
	)
}

type RootCommand struct {
	description    *command.BaseCommandDescription
	commandOptions *options.RootOptions
	cobraCommand   *cobra.Command
	logger         *zap.SugaredLogger
	logLevelSetter zap.AtomicLevel
}

func NewRootCommand(
	description *command.BaseCommandDescription,
	logLevelSetter zap.AtomicLevel,
	logger *zap.SugaredLogger,
	commandOptions *options.RootOptions,
) command.Command {
	return &RootCommand{
		description:    description,
		commandOptions: commandOptions,
		logger:         logger,
		logLevelSetter: logLevelSetter,
	}
}

func (r *RootCommand) ToCobraCommand() *cobra.Command {
	if r.cobraCommand != nil {
		return r.cobraCommand
	}

	r.cobraCommand = &cobra.Command{
		Use:   r.description.GetUse(),
		Short: r.description.GetShortDescription(),
		Long:  r.description.GetLongDescription(),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logLevel := "info"
			if options.RootOptionsInstance.Verbose {
				logLevel = "debug"
			}

			lvc, err := zapcore.ParseLevel(logLevel)
			if err != nil {
				r.logger.Warn("Failed to set level")
				return err
			}
			r.logLevelSetter.SetLevel(lvc)

			zap.S().Debugf("Current logging level enabled: %s", logLevel)

			return nil
		},
		// hide --completion for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
		RunE:         cli.RequireSubcommand,
	}

	r.cobraCommand.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	r.cobraCommand.Flags().SortFlags = false
	r.cobraCommand.PersistentFlags().SortFlags = false

	r.cobraCommand.SetOutput(color.Output)

	//	defer func() {
	//		// NOTE(shmel1k@): does not work. Sync will happen after end of the function
	//		_ = r.logger.Sync()
	//	}()

	r.cobraCommand.SetUsageTemplate(iCli.UsageTemplate)

	return r.cobraCommand
}

func (r *RootCommand) RegisterSubcommands(c ...command.Command) {
	// TODO(shmel1k@): add BaseCommand in order no to copypaste this method.
	for _, v := range c {
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}

var RootCmd *cobra.Command

func InitRootCmd(logLevelSetter zap.AtomicLevel, logger *zap.SugaredLogger) {
	rootCmd := NewRootCommand(
		command.NewDescription(
			"ydbops",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
		),
		logLevelSetter,
		logger,
		nil,
	)

	RootCmd = rootCmd.ToCobraCommand()
	RootCmd.AddCommand(NewRestartCmd())
}
