package options

import (
	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/internal/collections"
	"go.uber.org/multierr"
)

// AdditionalFlag allows adding extra flags not defined in command
//
// This behavior can be used if some additional parameters are added outside of some Option
//
//	TODO(shmel1k@): improve comment
type AdditionalFlag func(fs *pflag.FlagSet)

// Options is an interface to defile options flags and validation logic
type Options interface {
	DefineFlags(fs *pflag.FlagSet)
	Validate() error
}

func Validate(options ...Options) error {
	return multierr.Combine(
		collections.Convert(options, func(o Options) error { return o.Validate() })...,
	)
}
