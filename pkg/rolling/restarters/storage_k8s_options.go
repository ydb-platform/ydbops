package restarters

import (
	"github.com/spf13/pflag"
)

type StorageK8sOpts struct {
	k8sOpts
}

func (o *StorageK8sOpts) DefineFlags(fs *pflag.FlagSet) {
	o.k8sOpts.DefineFlags(fs)
}

func (o *StorageK8sOpts) Validate() error {
	if err := o.k8sOpts.Validate(); err != nil {
		return err
	}
	return nil
}
