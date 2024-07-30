package drop

import (
	"fmt"

	"github.com/spf13/pflag"

	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

type Options struct {
	TaskID string
}

func (o *Options) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TaskID, "task-id", "",
		"ID of your maintenance task (result of `ydbops maintenance host`)")
}

func (o *Options) Validate() error {
	if o.TaskID == "" {
		return fmt.Errorf("--task-id unspecified, argument required")
	}
	return nil
}

func (o *Options) Run(f cmdutil.Factory) error {
	return f.GetCMSClient().DropTask(o.TaskID)
}
