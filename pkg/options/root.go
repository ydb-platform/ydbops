package options

import (
	"github.com/spf13/pflag"
)

const (
	DefaultAuthType = "none"
	GRPCDefaultPort = 2135
)

type RootOptions struct {
	Auth     AuthOptions
	GRPC     GRPC
	Endpoint string
	Verbose  bool
}

var RootOptionsInstance = &RootOptions{}

func (o *RootOptions) Validate() error {
	if err := o.GRPC.Validate(); err != nil {
		return err
	}
	if err := o.Auth.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *RootOptions) DefineFlags(fs *pflag.FlagSet) {
	fs.BoolVar(&o.Verbose, "verbose", false, "TODO should enable verbose output")

	o.Auth.DefineFlags(fs)
	o.GRPC.DefineFlags(fs)
}
