package run

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

type RunOptions struct {
	*command.BaseOptions
	*rolling.RollingRestartOptions
	PayloadFilePath string
}

func (r *RunOptions) DefineFlags(fs *pflag.FlagSet) {
	r.RollingRestartOptions.DefineFlags(fs)
	fs.StringVar(
		&r.PayloadFilePath,
		"payload",
		"",
		"File path to arbitrary executable to run in the context of the local machine",
	)
}

func (r *RunOptions) Validate() error {
	err := r.RollingRestartOptions.Validate()
	if err != nil {
		return err
	}

	if r.PayloadFilePath == "" {
		return fmt.Errorf("empty --payload specified")
	}
	fileInfo, err := os.Stat(r.PayloadFilePath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("payload file '%s' does not exist", r.PayloadFilePath)
	}

	// Apologies, this is really an idiomatic way to check the permission in Go.
	// Just run some bitmagic. 0100 is octal, in binary it would be equivalent to:
	// 000001000000
	//   drwxrwxrwx
	executableByOwner := 0o100
	if fileInfo.Mode()&fs.FileMode(executableByOwner) != fs.FileMode(executableByOwner) {
		return fmt.Errorf("payload file '%s' is not executable by the owner", r.PayloadFilePath)
	}

	return nil
}
