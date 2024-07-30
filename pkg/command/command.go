package command

import (
	"errors"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/options"
)

type Description struct {
	use              string
	shortDescription string
	longDescription  string
}

type BaseOptions struct {
	Auth          options.AuthOptions
	GRPC          options.GRPC
	Verbose       bool
	ProfileFile   string
	ActiveProfile string
}

func (o *BaseOptions) Validate() error {
	if err := o.GRPC.Validate(); err != nil {
		return err
	}
	if err := o.Auth.Validate(); err != nil {
		return err
	}

	return nil
}

func (o *BaseOptions) DefineFlags(fs *pflag.FlagSet) {
	o.GRPC.DefineFlags(fs)
	o.Auth.DefineFlags(fs)

	fs.StringVar(
		&o.ActiveProfile, "profile",
		"",
		"Override currently set profile name from --config-file")

	defaultProfileLocation := ""
	if home, present := os.LookupEnv("HOME"); present {
		defaultProfileLocation = filepath.Join(home, "ydb", "ydbops", "config", "config.yaml")
	}

	_, err := os.Stat(defaultProfileLocation)
	if errors.Is(err, os.ErrNotExist) {
		// it is of course allowed, user does not have the default config,
		// "" will be treated as unspecified in profile code later
		defaultProfileLocation = ""
	}

	fs.StringVar(
		&o.ProfileFile, "profile-file",
		defaultProfileLocation,
		"Path to config file with profile data in yaml format. Default: $HOME/ydb/ydbops/config/config.yaml")

	fs.BoolVar(&o.Verbose, "verbose", false, "Switches log level from INFO to DEBUG")
}

func NewDescription(use, shortDescription, longDescription string) *Description {
	return &Description{
		use:              use,
		shortDescription: shortDescription,
		longDescription:  longDescription,
	}
}

func (b *Description) GetUse() string {
	return b.use
}

func (b *Description) GetShortDescription() string {
	return b.shortDescription
}

func (b *Description) GetLongDescription() string {
	return b.longDescription
}
