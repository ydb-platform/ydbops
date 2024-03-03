package run

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/pflag"
)

type Opts struct {
	PayloadFilepath string
}

func (o *Opts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&o.PayloadFilepath,
		"payload",
		"",
		"file with arbitrary shell code to run in the context of the machine, executing 'ydb-ops restart'.",
	)
}

func (o *Opts) Validate() error {
	if o.PayloadFilepath == "" {
		return fmt.Errorf("empty --payload specified")
	}
	if _, err := os.Stat(o.PayloadFilepath); errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("file '%s' does not exist", o.PayloadFilepath)
	}

	return nil
}
