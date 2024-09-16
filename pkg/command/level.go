package command

import (
	"go.uber.org/zap/zapcore"
)

type VerbosityLevel struct {
	level zapcore.Level
}

func (v *VerbosityLevel) String() string {
	return "false"
}

func (v *VerbosityLevel) Set(_ string) error {
	if v.level != zapcore.DebugLevel {
		v.level--
	}
	return nil
}

func (v *VerbosityLevel) Type() string {
	return "bool"
}

func (v *VerbosityLevel) Level() zapcore.Level {
	return v.level
}

func newVerbosityLevel() VerbosityLevel {
	return VerbosityLevel{
		level: zapcore.WarnLevel,
	}
}
