package maintenance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewCreateCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	maintenanceCreateOpts := &options.MaintenanceCreateOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "create",
		Short: "Create a maintenance task to obtain a set of hosts",
		Long: `ydbops maintenance create: 
  Create a maintenance task, which allows taking the set of hosts out of the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			maintenanceCreateOpts, rootOpts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.InitConnectionFactory(
				*rootOpts,
				options.Logger,
				options.DefaultRetryCount,
			)
			if err != nil {
				return err
			}

			taskId, err := maintenance.CreateTask(maintenanceCreateOpts)
			if err != nil {
				return err
			}

			fmt.Printf(
				"Your task id is:\n\n%s\n\nPlease write it down for refreshing and completing the task later.\n",
				taskId,
			)

			return nil
		},
	})

	maintenanceCreateOpts.DefineFlags(cmd.PersistentFlags())
	options.RootOptionsInstance.DefineFlags(cmd.PersistentFlags())

	return cmd
}

func init() {
}
