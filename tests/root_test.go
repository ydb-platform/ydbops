package tests

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ydb-platform/ydb-ops/cmd"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func createTestingLogger(level string) (zap.AtomicLevel, *zap.Logger, *observer.ObservedLogs) {
	atom, _ := zap.ParseAtomicLevel(level)
	core, logs := observer.New(zap.InfoLevel)
	return atom, zap.New(core), logs
}

func TestYdbOpsHelp(t *testing.T) {
	actual := new(bytes.Buffer)
	logLevelSetter, logger, _ := createTestingLogger("info")
	cmd.InitRootCmd(logLevelSetter, logger)
	cmd.RootCmd.SetOut(actual)
	cmd.RootCmd.SetErr(actual)
	cmd.RootCmd.SetArgs([]string{})
	_ = cmd.RootCmd.Execute()

	expected := `TODO ydb-ops long description

Usage:
  ydb-ops [command]

Available Commands:
  help        Help about any command
  restart     restart short description

Flags:
      --auth-env-name string       Authentication environment variable name (type: env) (default "YDB_TOKEN")
      --auth-file-token string     Authentication file token name (type: file)
      --auth-iam-endpoint string   Authentication iam endpoint (type: iam) (default "iam.api.cloud.yandex.net")
      --auth-iam-key-file string   Authentication iam key file path (type: iam)
      --auth-type string           Authentication types: [none env file iam] (default "none")
      --ca-file string             TODO path to root ca file
      --endpoint string            TODO GRPC addresses which will be used to connect to cluster
  -h, --help                       help for ydb-ops
      --verbose                    TODO should enable verbose output

Use "ydb-ops [command] --help" for more information about a command.
`

	assert.Equal(t, actual.String(), expected, "actual is not expected")
}
