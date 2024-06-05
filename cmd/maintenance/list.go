package maintenance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewListCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "list",
		Short: "List all existing maintenance tasks",
		Long: `ydbops maintenance list: 
  List all existing maintenance tasks on the cluster.
  Can be useful if you lost your task id to refresh/complete your own task.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			rootOpts,
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

			taskIDs, err := maintenance.ListTasks()

			for _, taskID := range taskIDs {
				fmt.Printf("%s\n", taskID)
			}

			if err != nil {
				return err
			}
			return nil
		},
	})

	options.RootOptionsInstance.DefineFlags(cmd.PersistentFlags())

	return cmd
}

func init() {
}
