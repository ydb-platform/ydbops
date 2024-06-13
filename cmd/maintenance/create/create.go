package create

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client/cms"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

func New(f cmdutil.Factory) *cobra.Command {
	opts := &Options{}

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
			taskId, err := f.GetCMSClient().CreateMaintenanceTask(cms.MaintenanceTaskParams{
				Hosts:            opts.HostFQDNs,
				Duration:         opts.GetMaintenanceDuration(),
				AvailabilityMode: opts.GetAvailabilityMode(),
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
