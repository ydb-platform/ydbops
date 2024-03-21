package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ydb-platform/ydbops/internal/cli"
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
	)
}

var RootCmd *cobra.Command

func InitRootCmd(logLevelSetter zap.AtomicLevel, logger *zap.SugaredLogger) {
	RootCmd = &cobra.Command{
		Use:   "ydbops",
		Short: "ydbops: a CLI tool with common YDB cluster maintenance operations",
		Long:  "ydbops: a CLI tool with common YDB cluster maintenance operations",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			logLevel := "info"
			if options.RootOptionsInstance.Verbose {
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
		// TODO decide if we need to hide this, for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
		RunE:         cli.RequireSubcommand,
	}

	RootCmd.SetHelpCommand(&cobra.Command{Hidden: true})

	RootCmd.SetOutput(color.Output)

	defer func() {
		_ = logger.Sync()
	}()

	options.Logger = logger

	options.RootOptionsInstance.DefineFlags(RootCmd.PersistentFlags())

	registerAllSubcommands(RootCmd)

	RootCmd.SetUsageTemplate(cli.UsageTemplate)
}
