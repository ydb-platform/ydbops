package main

import (
	"os"

	"github.com/ydb-platform/ydbops/cmd"
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
	baseOptions     *command.BaseOptions
	cmsClient       cms.Client
	discoveryClient discovery.Client
)

func initFactory() {
	factory = cmdutil.New(baseOptions, cmsClient, discoveryClient)
}

func initClients(
	cf connectionsfactory.Factory,
	logger *zap.SugaredLogger,
	cp credentials.Provider,
) {
	cmsClient = cms.NewCMSClient(cf, logger, cp)
	discoveryClient = discovery.NewDiscoveryClient(cf, logger, cp)
}

func main() {
	logLevelSetter, logger := createLogger("info")
	baseOptions = &command.BaseOptions{}
	root := cmd.NewRootCommand(logLevelSetter, logger.Sugar(), baseOptions)
	cf := connectionsfactory.New(baseOptions)

	options.Logger = logger.Sugar() // TODO(shmel1k@): tmp hack

	credentialsProvider := credentials.New(baseOptions, cf, logger.Sugar(), nil)
	initClients(cf, logger.Sugar(), credentialsProvider)
	initFactory()

	defer func() {
		_ = logger.Sync()
	}()
	cmd.InitRootCommandTree(root, factory)
	_ = root.Execute()
}
