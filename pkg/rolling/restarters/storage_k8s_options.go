package restarters

import (
	"fmt"
	"github.com/spf13/pflag"
)

type StorageK8sOpts struct {
	k8sOpts
	storage        string
}

func (o *StorageK8sOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.storage, "storage", "", "Storage resource name to restart")
}

func (o *StorageK8sOpts) Validate() error {
	if err := o.k8sOpts.Validate(); err != nil {
		return err
	}
	if o.storage == "" {
		return fmt.Errorf("Please specify a non-empty --storage.")
	}
	return nil
}
