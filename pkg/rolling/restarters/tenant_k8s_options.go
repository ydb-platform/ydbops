package restarters

import (
	"fmt"
	"github.com/spf13/pflag"
)

type TenantK8sOpts struct {
	k8sOpts
	database string
}

func (o *TenantK8sOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.database, "database", "", "Database resource name to restart")
}

func (o *TenantK8sOpts) Validate() error {
	if err := o.k8sOpts.Validate(); err != nil {
		return err
	}
	if o.database == "" {
		return fmt.Errorf("Please specify a non-empty --database.")
	}
	return nil
}
