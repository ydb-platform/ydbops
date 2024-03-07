package restarters

import (
	"github.com/spf13/pflag"
)

type StorageBaremetalOpts struct {
	baremetalOpts
	kikimrStorageUnit bool
}

func (o *StorageBaremetalOpts) DefineFlags(fs *pflag.FlagSet) {
	o.baremetalOpts.DefineFlags(fs)
	fs.BoolVar(&o.kikimrStorageUnit, "kikimr", false, "Use 'kikimr' as the storage unit name to restart")
}

func (o *StorageBaremetalOpts) Validate() error {
	if err := o.baremetalOpts.Validate(); err != nil {
		return err
	}
	return nil
}
