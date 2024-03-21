package options

import (
	"github.com/spf13/pflag"
)

type RootOptions struct {
	Auth           AuthOptions
	GRPC           GRPC
	Verbose        bool
	KubeconfigPath string
	K8sNamespace   string
	ProfileFile    string
	ActiveProfile  string
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
	o.GRPC.DefineFlags(fs)
	o.Auth.DefineFlags(fs)

	fs.StringVar(
		&o.ProfileFile, "config-file",
		"",
		"Path to config file with profile data in yaml format")

	fs.StringVar(
		&o.ActiveProfile, "profile",
		"",
		"Override currently set profile name from --config-file")

	fs.BoolVar(&o.Verbose, "verbose", false, "Switches log level from INFO to DEBUG")
}
