package refresh

import (
	"fmt"

	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
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
	task, err := f.GetCMSClient().RefreshTask(o.TaskID)
	if err != nil {
		return err
	}

	fmt.Println(prettyprint.TaskToString(task))

	return nil
}
