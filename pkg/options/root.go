package options

import (
	"github.com/spf13/pflag"
)

type RootOptions struct {
	Auth     AuthOptions
	GRPC     GRPC
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
	o.Auth.DefineFlags(fs)
	o.GRPC.DefineFlags(fs)

	fs.BoolVar(&o.Verbose, "verbose", false, "Switches log level from INFO to DEBUG")
}
