package command

import (
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
		&o.ProfileFile, "config-file",
		"",
		"Path to config file with profile data in yaml format")

	fs.StringVar(
		&o.ActiveProfile, "profile",
		"",
		"Override currently set profile name from --config-file")

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
