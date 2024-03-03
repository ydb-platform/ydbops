package storage_baremetal

import (
	"github.com/spf13/pflag"
)

type Opts struct {
	SSHArgs []string
	IsOldSystemdKikimr bool
}

func (o *Opts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVarP(&o.SSHArgs, "ssh-args", "", nil, "TODO SSH command arguments")
	fs.BoolVar(&o.IsOldSystemdKikimr, "kikimr", false, "TODO kikimr instead of ydb-server-storage.service")
}

func (o *Opts) Validate() error {
	return nil
}
