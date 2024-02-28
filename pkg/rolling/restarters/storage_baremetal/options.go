package storage_baremetal

import (
	"github.com/spf13/pflag"
)

type opts struct {
	SSHArgs []string
}

func (o *opts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringArrayVarP(&o.SSHArgs, "ssh-args", "", nil, "TODO SSH command arguments")
}

func (o *opts) Validate() error {
	return nil
}
