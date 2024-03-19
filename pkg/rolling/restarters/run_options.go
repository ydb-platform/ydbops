package restarters

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/pflag"
)

type RunOpts struct {
	PayloadFilepath string
}

func (o *RunOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&o.PayloadFilepath,
		"payload",
		"",
		"File path to arbitrary executable to run in the context of the local machine",
	)
}

func (o *RunOpts) Validate() error {
	if o.PayloadFilepath == "" {
		return fmt.Errorf("empty --payload specified")
	}
	fileInfo, err := os.Stat(o.PayloadFilepath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("payload file '%s' does not exist", o.PayloadFilepath)
	}

	// Apologies, this is really an idiomatic way to check the permission in Go.
	// Just run some bitmagic. 0100 is octal, in binary it would be equivalent to:
	// 000001000000
	//   drwxrwxrwx
	executableByOwner := 0o100
	if fileInfo.Mode()&fs.FileMode(executableByOwner) != fs.FileMode(executableByOwner) {
		return fmt.Errorf("payload file '%s' is not executable by the owner", o.PayloadFilepath)
	}

	return nil
}
