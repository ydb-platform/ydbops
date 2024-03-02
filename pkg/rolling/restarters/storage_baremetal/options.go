package storage_baremetal

import (
	"github.com/spf13/pflag"
)

type Opts struct {
	SSHArgs []string
}

func (o *Opts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVarP(&o.SSHArgs, "ssh-args", "", nil, "TODO SSH command arguments")
}

func (o *Opts) Validate() error {
	return nil
}
