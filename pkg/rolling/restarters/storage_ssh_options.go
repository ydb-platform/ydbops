package restarters

import (
	"github.com/spf13/pflag"
)

type StorageSSHOpts struct {
	sshOpts
	kikimrStorageUnit bool
}

func (o *StorageSSHOpts) DefineFlags(fs *pflag.FlagSet) {
	o.sshOpts.DefineFlags(fs)
	fs.BoolVar(&o.kikimrStorageUnit, "kikimr", false, "Use 'kikimr' as the storage unit name to restart")
}

func (o *StorageSSHOpts) Validate() error {
	if err := o.sshOpts.Validate(); err != nil {
		return err
	}
	return nil
}
