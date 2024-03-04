package restarters

import (
	"github.com/spf13/pflag"
)

type StorageBaremetalOpts struct {
	SSHArgs []string
	IsOldSystemdKikimr bool
}

func (o *StorageBaremetalOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVarP(&o.SSHArgs, "ssh-args", "", nil, "TODO SSH command arguments")
	fs.BoolVar(&o.IsOldSystemdKikimr, "kikimr", false, "TODO use")
}

func (o *StorageBaremetalOpts) Validate() error {
	return nil
}
