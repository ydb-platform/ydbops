package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/cmd/restart"
	"github.com/ydb-platform/ydb-ops/cmd/restart/storage"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func registerAllSubcommands(root *cobra.Command) {
	k8sCmd := storage.NewK8sCmd()
	storageCmd := restart.NewStorageCmd()
	restartCmd := NewRestartCmd()

	storageCmd.AddCommand(k8sCmd)
	restartCmd.AddCommand(storageCmd)
	root.AddCommand(restartCmd)
}

func registerRootOptions(root *cobra.Command) {
	options.RootOptionsInstance.DefineFlags(root.PersistentFlags())
}

var RootCmd *cobra.Command

func InitRootCmd(logLevelSetter zap.AtomicLevel, logger *zap.Logger) {
	RootCmd = &cobra.Command{
		Use:   "ydb-ops",
		Short: "TODO ydb-ops short description",
		Long:  "TODO ydb-ops long description",
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

			logger.Debug(fmt.Sprintf("Current logging level enabled: %s", logLevel))

			return options.RootOptionsInstance.Validate()
		},
		// TODO decide if we need to hide this, for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	defer util.IgnoreError(logger.Sync)

	options.Logger = logger
	registerRootOptions(RootCmd)
	registerAllSubcommands(RootCmd)
}
