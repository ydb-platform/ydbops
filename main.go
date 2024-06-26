package main

import (
	"os"

	"github.com/ydb-platform/ydbops/cmd"
	"github.com/ydb-platform/ydbops/cmd/maintenance"
	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client/auth/credentials"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/client/connectionsfactory"
	"github.com/ydb-platform/ydbops/pkg/client/discovery"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
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

var (
	factory         cmdutil.Factory
	cmsClient       cms.Client
	discoveryClient discovery.Client
)

func initFactory() {
	factory = cmdutil.New(cmsClient, discoveryClient)
}

func initClients(
	cf connectionsfactory.Factory,
	logger *zap.SugaredLogger,
	cp credentials.Provider,
) {
	cmsClient = cms.NewCMSClient(cf, logger, cp)
	discoveryClient = discovery.NewDiscoveryClient(cf, logger, cp)
}

func initCommandTree(rootOptions *command.BaseOptions, logLevelSetter zap.AtomicLevel, logger *zap.SugaredLogger) (root command.Command) {
	baseCommand := command.NewBase(rootOptions)
	root = cmd.NewRootCommand(
		command.NewDescription(
			"ydbops",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
			"ydbops: a CLI tool for performing YDB cluster maintenance operations",
		),
		baseCommand,
		logLevelSetter,
		logger,
	)
	root.RegisterOptions()
	rootOptions.DefineFlags(root.ToCobraCommand().PersistentFlags())

	restartCommand := cmd.NewRestartCommand(
		cmd.RestartCommandDescription,
		baseCommand,
		factory,
	)
	root.RegisterSubcommands(restartCommand)

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
		baseCommand,
		factory,
	)
	root.RegisterSubcommands(runCommand)

	hostCommand := maintenance.NewHostCommand(baseCommand)
	maintenanceCommand := cmd.NewMaintenanceCommand(baseCommand)
	maintenanceCommand.RegisterSubcommands(hostCommand)
	root.RegisterSubcommands(maintenanceCommand)

	cli.SetDefaultsOn(root.ToCobraCommand())
	root.ToCobraCommand().SetUsageTemplate(iCli.UsageTemplate)
	return root
}

func main() {
	rootOptions := &command.BaseOptions{}
	cf := connectionsfactory.New(rootOptions)
	logLevelSetter, logger := createLogger("info")

	options.Logger = logger.Sugar() // TODO(shmel1k@): tmp hack

	credentialsProvider := credentials.New(rootOptions, cf, logger.Sugar(), nil)
	initClients(cf, logger.Sugar(), credentialsProvider)
	initFactory()

	root := initCommandTree(rootOptions, logLevelSetter, logger.Sugar())
	defer func() {
		_ = logger.Sync()
	}()
	_ = root.ToCobraCommand().Execute()
}
