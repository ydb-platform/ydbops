package options

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/pflag"
)

type RootOptions struct {
	Auth          AuthOptions
	GRPC          GRPC
	Verbose       bool
	ProfileFile   string
	ActiveProfile string
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

	defaultProfileLocation := ""
	if home, present := os.LookupEnv("HOME"); present {
		defaultProfileLocation = fmt.Sprintf("%s/ydb/ydbops/config/config.yaml", home)
	}

	_, err := os.Stat(defaultProfileLocation)
	if errors.Is(err, os.ErrNotExist) {
		// it is of course allowed, user does not have the default config,
		// "" will be treated as unspecified in profile code later
		defaultProfileLocation = ""
	}

	fs.StringVar(
		&o.ProfileFile, "config-file",
		defaultProfileLocation,
		"Path to config file with profile data in yaml format. Default: $HOME/ydb/ydbops/config/config.yaml")

	fs.StringVar(
		&o.ActiveProfile, "profile",
		"",
		"Override currently set profile name from --config-file")

	fs.BoolVar(&o.Verbose, "verbose", false, "Switches log level from INFO to DEBUG")
}
