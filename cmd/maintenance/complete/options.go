package complete

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

type Options struct {
	TaskID string
	Hosts  []string
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	fs.StringSliceVar(&o.Hosts, "hosts", []string{},
		`FQDNs or nodeIds of hosts with completed maintenance. You can specify a list of host FQDNs or a list of node ids,
  but you can not mix host FQDNs and node ids in this option. The list is comma-delimited.
  E.g.: '--hosts=1,2,3' or '--hosts=fqdn1,fqdn2,fqdn3'`)
	fs.StringVar(&o.TaskID, "task-id", "",
		"ID of your maintenance task (result of `ydbops maintenance host`)")
}

func (o *Options) Validate() error {
	// TODO(shmel1k@): remove copypaste between drop, create & refresh methods.
	if len(o.Hosts) == 0 {
		return fmt.Errorf("--hosts unspecified")
	}
	if o.TaskID == "" {
		return fmt.Errorf("--task-id unspecified, argument required")
	}
	return nil
}

func (o *Options) Run(f cmdutil.Factory) error {
	result, err := f.GetCMSClient().CompleteActions(o.TaskID, o.Hosts)
	if err != nil {
		return err
	}

	fmt.Println(prettyprint.ResultToString(result))
	return nil
}
