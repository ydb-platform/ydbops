package restarters

import (
	"github.com/spf13/pflag"
)

type TenantSSHOpts struct {
	sshOpts
	kikimrTenantUnit bool
}

func (o *TenantSSHOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.kikimrTenantUnit, "kikimr", false, "Use 'kikimr-multi@{ic_port}' as the dynnode unit name to restart")
	o.sshOpts.DefineFlags(fs)
}

func (o *TenantSSHOpts) Validate() error {
	if err := o.sshOpts.Validate(); err != nil {
		return err
	}
	return nil
}
