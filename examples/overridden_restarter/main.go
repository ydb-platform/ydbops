package main

import (
	"os"

	"github.com/ydb-platform/ydbops/cmd"
	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
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

func initCommandTree(opts *command.BaseOptions, logLevelSetter zap.AtomicLevel, logger *zap.SugaredLogger) (root command.Command) {
	root = cmd.NewRootCommand(
		command.NewDescription(
			"ydbops",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
		),
		logLevelSetter,
		logger,
		opts,
	)
	root.RegisterOptions(opts)
	opts.DefineFlags(root.ToCobraCommand(opts).PersistentFlags())

	restartCommand := cmd.NewRestartCommand(
		cmd.RestartCommandDescription,
		cli.PopulateProfileDefaultsAndValidate,
	)
	root.RegisterSubcommands(opts, restartCommand)
	cli.SetDefaultsOn(root.ToCobraCommand(opts))

	restarter := restarters.NewRunRestarter(logger)
	runCommand := cmd.NewRunCommand(
		command.NewDescription(
			"run",
			"Run an arbitrary executable (e.g. shell code) in the context of the local machine",
			`ydbops restart run:
		Run an arbitrary executable (e.g. shell code) in the context of the local machine
		(where rolling-restart is launched). For example, if you want to execute ssh commands
		on the ydb cluster node, you must write ssh commands yourself. See the examples.

		For every node released by CMS, ydbops will execute this payload independently.

		Restart will be treated as successful if your executable finished with a zero
		return code.

		Certain environment variable will be passed to your executable on each run:
			$HOSTNAME: the fqdn of the node currently released by CMS.`,
		),
		restarter,
	)
	root.RegisterSubcommands(opts, runCommand)

	root.ToCobraCommand(opts).SetUsageTemplate(iCli.UsageTemplate)
	return root
}

func main() {
	opts := &command.BaseOptions{}
	logLevelSetter, logger := createLogger("info")
	root := initCommandTree(opts, logLevelSetter, logger.Sugar())
	defer func() {
		_ = logger.Sync()
	}()
	_ = root.ToCobraCommand(opts).Execute()
}
