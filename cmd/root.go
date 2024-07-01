package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

type RootCommand struct {
	description    *command.Description
	cobraCommand   *cobra.Command
	logger         *zap.SugaredLogger
	logLevelSetter zap.AtomicLevel
}

type RootOptions struct {
	*command.BaseOptions
}

func (r *RootOptions) Validate() error {
	return r.BaseOptions.Validate()
}

func (r *RootOptions) DefineFlags(fs *pflag.FlagSet) {
	r.BaseOptions.DefineFlags(fs)
}

func NewRootCommand(
	description *command.Description,
	logLevelSetter zap.AtomicLevel,
	logger *zap.SugaredLogger,
) *cobra.Command {
	roptions := &RootOptions{}
	cmd := &cobra.Command{
		Use:   description.GetUse(),
		Short: description.GetShortDescription(),
		Long:  description.GetLongDescription(),
		// hide --completion for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			roptions.DefineFlags(cmd.PersistentFlags())
			err := options.Validate()
			if err != nil {
				return err
			}

			logLevel := "info"
			if roptions.Verbose {
				logLevel = "debug"
			}

			lvc, err := zapcore.ParseLevel(logLevel)
			if err != nil {
				logger.Warn("Failed to set level")
				return err
			}
			logLevelSetter.SetLevel(lvc)

			zap.S().Debugf("Current logging level enabled: %s", logLevel)
			return nil
		},
		RunE: cli.RequireSubcommand,
	}

	cmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.SetOutput(color.Output)
	cmd.SetUsageTemplate(iCli.UsageTemplate)

	return cmd
}
