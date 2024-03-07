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

	expected := ``

	assert.Equal(t, actual.String(), expected, "actual is not expected")
}
