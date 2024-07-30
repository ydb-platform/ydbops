package list

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

func New(f cmdutil.Factory) *cobra.Command {
	_ = &Options{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "list",
		Short: "List all existing maintenance tasks",
		Long: `ydbops maintenance list:
  List all existing maintenance tasks on the cluster.
  Can be useful if you lost your task id to refresh/complete your own task.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			userSID, err := f.GetDiscoveryClient().WhoAmI()
			if err != nil {
				return err
			}

			tasks, err := f.GetCMSClient().MaintenanceTasks(userSID)
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("There are no maintenance tasks, associated with your user, at the moment.")
			} else {
				for _, task := range tasks {
					taskInfo := prettyprint.TaskToString(task)
					fmt.Println(taskInfo)
				}
			}

			return nil
		},
	})

	return cmd
}
