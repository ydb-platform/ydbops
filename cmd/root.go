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
	description    *command.Description
	cobraCommand   *cobra.Command
	logger         *zap.SugaredLogger
	logLevelSetter zap.AtomicLevel
	opts           *command.BaseOptions
}

func NewRootCommand(
	description *command.Description,
	logLevelSetter zap.AtomicLevel,
	logger *zap.SugaredLogger,
	opts *command.BaseOptions,
) command.Command {
	return &RootCommand{
		description:    description,
		logger:         logger,
		logLevelSetter: logLevelSetter,
		opts:           opts,
	}
}

func (r *RootCommand) RunCallback(opts *command.BaseOptions) func(cmd *cobra.Command, args []string) error {
	// TODO(shmel1k@): nil nil?
	return cli.RequireSubcommand
}

func (r *RootCommand) RegisterOptions(opts *command.BaseOptions) {
}

func (r *RootCommand) ToCobraCommand(opts *command.BaseOptions) *cobra.Command {
	if r.cobraCommand != nil {
		return r.cobraCommand
	}

	r.cobraCommand = &cobra.Command{
		Use:   r.description.GetUse(),
		Short: r.description.GetShortDescription(),
		Long:  r.description.GetLongDescription(),
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logLevel := "info"
			if opts.Verbose {
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
		RunE:         r.RunCallback(opts),
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

func (r *RootCommand) RegisterSubcommands(opts *command.BaseOptions, c ...command.Command) {
	for _, v := range c {
		v.RegisterOptions(opts)
		cli.SetDefaultsOn(v.ToCobraCommand(opts))
		r.ToCobraCommand(opts).AddCommand(v.ToCobraCommand(opts))
	}
}
