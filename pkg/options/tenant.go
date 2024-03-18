package options

import "github.com/spf13/pflag"

type TenantOptions struct {
	Tenants []string
}

func (o *TenantOptions) Validate() error {
	return nil
}

func (o *TenantOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Tenants, "tenants", o.Tenants, "Include only specified tenants")
}
