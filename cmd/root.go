package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
)

type RootCommand struct {
	*command.Base
	description    *command.Description
	cobraCommand   *cobra.Command
	logger         *zap.SugaredLogger
	logLevelSetter zap.AtomicLevel
}

func NewRootCommand(
	description *command.Description,
	baseCommand *command.Base,
	logLevelSetter zap.AtomicLevel,
	logger *zap.SugaredLogger,
) command.Command {
	return &RootCommand{
		Base:           baseCommand,
		description:    description,
		logger:         logger,
		logLevelSetter: logLevelSetter,
	}
}

func (r *RootCommand) RunCallback() func(cmd *cobra.Command, args []string) error {
	return cli.RequireSubcommand
}

func (r *RootCommand) RegisterOptions() {
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
			if r.GetBaseOptions().Verbose {
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
		RunE:         r.RunCallback(),
	}

	r.cobraCommand.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})

	r.cobraCommand.Flags().SortFlags = false
	r.cobraCommand.PersistentFlags().SortFlags = false

	r.cobraCommand.SetOutput(color.Output)

	r.cobraCommand.SetUsageTemplate(iCli.UsageTemplate)

	return r.cobraCommand
}

func (r *RootCommand) RegisterSubcommands(c ...command.Command) {
	for _, v := range c {
		v.RegisterOptions()
		cli.SetDefaultsOn(v.ToCobraCommand())
		r.ToCobraCommand().AddCommand(v.ToCobraCommand())
	}
}
