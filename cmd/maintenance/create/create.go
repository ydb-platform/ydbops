package create

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

func New(f cmdutil.Factory) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "create",
		Short: "Create a maintenance task to obtain a set of hosts",
		Long: `ydbops maintenance create:
  Create a maintenance task, which allows taking the set of hosts out of the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(), opts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskUID := cms.TaskUuidPrefix + uuid.New().String()
			duration := time.Duration(opts.RestartOptions.RestartDuration) * time.Minute
			taskId, err := f.GetCMSClient().CreateMaintenanceTask(cms.MaintenanceTaskParams{
				Hosts:            opts.RestartOptions.Hosts,
				Duration:         durationpb.New(duration),
				AvailabilityMode: opts.RestartOptions.GetAvailabilityMode(),
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
		},
	})

	opts.DefineFlags(cmd.PersistentFlags())

	return cmd
}
