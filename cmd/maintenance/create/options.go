package create

import (
	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/rolling"
)

type Options struct {
	*rolling.RestartOptions
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	o.RestartOptions.DefineFlags(fs)
}

func (o *Options) Validate() error {
	return o.RestartOptions.Validate()
}
