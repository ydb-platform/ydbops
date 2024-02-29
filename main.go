package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/cmd"
	"github.com/ydb-platform/ydb-ops/cmd/restart"
	"github.com/ydb-platform/ydb-ops/cmd/restart/storage"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func createLogger(level string) (zap.AtomicLevel, *zap.Logger) {
	atom, _ := zap.ParseAtomicLevel(level)
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	logger := zap.New(
		zapcore.NewCore(
			zapcore.NewConsoleEncoder(encoderCfg),
			zapcore.Lock(os.Stdout),
			atom,
		),
	)

	_ = zap.ReplaceGlobals(logger)
	return atom, logger
}

func registerAllSubcommands(root *cobra.Command) {
	k8sCmd := storage.NewK8sCmd()
	storageCmd := restart.NewStorageCmd()
	restartCmd := cmd.NewRestartCmd()

	storageCmd.AddCommand(k8sCmd)
	restartCmd.AddCommand(storageCmd)
	root.AddCommand(restartCmd)
}

func registerRootOptions(root *cobra.Command) {
	options.RootOptionsInstance.DefineFlags(root.PersistentFlags())
}

func main() {
	logLevelSetter, logger := createLogger("info")
	root := &cobra.Command{
		Use:   "ydb-ops",
		Short: "TODO ydb-ops short description",
		Long:  "TODO ydb-ops long description",
		PersistentPreRunE: func(_ *cobra.Command, _ []string) error {
			fmt.Println("some debug info from PersistentPreRunE")
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

			return options.RootOptionsInstance.Validate()
		},
		// Run: func(_ *cobra.Command, _ []string) {
		// },
		// TODO decide if we need to hide this, for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	defer util.IgnoreError(logger.Sync)

	options.Logger = logger
	registerRootOptions(root)
	registerAllSubcommands(root)

	if err := root.Execute(); err != nil {
		logger.Fatal("failed to execute restart", zap.Error(err))
	}
}
