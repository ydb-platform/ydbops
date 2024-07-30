package run

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"github.com/ydb-platform/ydbops/pkg/rolling/restarters"
	"go.uber.org/zap"
)

type Options struct {
	*rolling.RestartOptions
	PayloadFilePath string
}

func (r *Options) DefineFlags(fs *pflag.FlagSet) {
	r.RestartOptions.DefineFlags(fs)
	fs.StringVar(
		&r.PayloadFilePath,
		"payload",
		"",
		"File path to arbitrary executable to run in the context of the local machine",
	)
}

func (r *Options) Validate() error {
	err := r.RestartOptions.Validate()
	if err != nil {
		return err
	}

	if r.PayloadFilePath == "" {
		return fmt.Errorf("empty --payload specified")
	}
	fileInfo, err := os.Stat(r.PayloadFilePath)
	if errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("payload file '%s' does not exist", r.PayloadFilePath)
	}

	// Apologies, this is really an idiomatic way to check the permission in Go.
	// Just run some bitmagic. 0100 is octal, in binary it would be equivalent to:
	// 000001000000
	//   drwxrwxrwx
	executableByOwner := 0o100
	if fileInfo.Mode()&fs.FileMode(executableByOwner) != fs.FileMode(executableByOwner) {
		return fmt.Errorf("payload file '%s' is not executable by the owner", r.PayloadFilePath)
	}

	return nil
}

func (r *Options) Run(f cmdutil.Factory) error {
	bothUnspecified := !r.RestartOptions.Storage && !r.RestartOptions.Tenant

	restarter := restarters.NewRunRestarter(zap.S(), &restarters.RunRestarterParams{
		PayloadFilePath: r.PayloadFilePath,
	})

	var executer rolling.Executer
	var err error
	if r.RestartOptions.Storage || bothUnspecified {
		restarter.SetStorageOnly()
		executer = rolling.NewExecuter(r.RestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
		err = executer.Execute()
	}

	if err == nil && (r.RestartOptions.Tenant || bothUnspecified) {
		restarter.SetDynnodeOnly()
		executer = rolling.NewExecuter(r.RestartOptions, options.Logger, f.GetCMSClient(), f.GetDiscoveryClient(), restarter)
		err = executer.Execute()
	}

	return err
}
