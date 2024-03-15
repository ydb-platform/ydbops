package main

import (
	"os"

	"github.com/ydb-platform/ydbops/cmd"
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

func main() {
	logLevelSetter, logger := createLogger("info")
  cmd.InitRootCmd(logLevelSetter, logger.Sugar())

	if err := cmd.RootCmd.Execute(); err != nil {
		logger.Fatal("failed to execute restart", zap.Error(err))
	}
}
