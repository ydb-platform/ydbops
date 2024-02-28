package main

import (
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

func main() {
	logLevel := "info"
	logLevelSetter, logger := createLogger(logLevel)
	root := &cobra.Command{
		Use:   "ydb-ops",
		Short: "TODO ydb-ops short description",
		Long:  "TODO ydb-ops long description",
		PersistentPreRun: func(_ *cobra.Command, _ []string) {
			lvc, err := zapcore.ParseLevel(logLevel)
			if err != nil {
				logger.Warn("Failed to set level")
				return
			}
			logLevelSetter.SetLevel(lvc)
		},
		// TODO decide if we need to hide this, for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
	}

	defer util.IgnoreError(logger.Sync)

	options.Logger = logger

	root.PersistentFlags().StringVarP(&logLevel, "log-level", "", logLevel, "Logging level")

	registerAllSubcommands(root)

	if err := root.Execute(); err != nil {
		logger.Fatal("failed to execute restart", zap.Error(err))
	}

}
