package complete

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

type Options struct {
	TaskID    string
	HostFQDNs []string
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.HostFQDNs, "hosts", []string{},
		"FQDNs of hosts with completed maintenance")
	fs.StringVar(&o.TaskID, "task-id", "",
		"ID of your maintenance task (result of `ydbops maintenance host`)")
}

func (o *Options) Validate() error {
	// TODO(shmel1k@): remove copypaste between drop, create & refresh methods.
	if len(o.HostFQDNs) == 0 {
		return fmt.Errorf("--hosts unspecified")
	}
	if o.TaskID == "" {
		return fmt.Errorf("--task-id unspecified, argument required")
	}
	return nil
}

func (o *Options) Run(f cmdutil.Factory) error {
	result, err := f.GetCMSClient().CompleteActions(o.TaskID, o.HostFQDNs)
	if err != nil {
		return err
	}

	fmt.Println(prettyprint.ResultToString(result))
	return nil
}
