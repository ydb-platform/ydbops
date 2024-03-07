package restarters

import (
	"github.com/spf13/pflag"
)

type TenantBaremetalOpts struct {
  baremetalOpts
	kikimrTenantUnit bool
}

func (o *TenantBaremetalOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.kikimrTenantUnit, "kikimr", false, "Use 'kikimr-multi@{ic_port}' as the dynnode unit name to restart")
  o.baremetalOpts.DefineFlags(fs)
}

func (o *TenantBaremetalOpts) Validate() error {
	if err := o.baremetalOpts.Validate(); err != nil {
		return err
	}
	return nil
}
