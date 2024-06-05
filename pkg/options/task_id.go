package options

import (
	"github.com/spf13/pflag"
)

type TaskIdOpts struct {
	TaskID string
}

func (o *TaskIdOpts) DefineFlags(fs *pflag.FlagSet) {
	fs.StringVar(&o.TaskID, "task-id", "",
		"ID of your maintenance task (result of `ydbops maintenance host`)")
}

func (o *TaskIdOpts) Validate() error {
	return nil
}
