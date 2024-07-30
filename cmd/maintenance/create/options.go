package create

import (
	"github.com/spf13/pflag"
	"github.com/ydb-platform/ydbops/pkg/rolling"
	"google.golang.org/protobuf/types/known/durationpb"
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

func (o *Options) Run(f cmdutil.Factory) error {
	taskUID := cms.TaskUuidPrefix + uuid.New().String()
	duration := time.Duration(o.RestartOptions.RestartDuration) * time.Minute
	taskId, err := f.GetCMSClient().CreateMaintenanceTask(cms.MaintenanceTaskParams{
		Hosts:            o.RestartOptions.Hosts,
		Duration:         durationpb.New(duration),
		AvailabilityMode: o.RestartOptions.GetAvailabilityMode(),
		ScopeType:        cms.HostScope,
		TaskUID:          taskUID,
	})
	if err != nil {
		return err
	}

	fmt.Printf(
		"Your task id is:\n\n%s\n\nPlease write it down for refreshing and completing the task later.\n",
		taskId.GetTaskUid(),
	)

	return nil
}
