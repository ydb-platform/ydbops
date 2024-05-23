package maintenance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/options"

	"github.com/ydb-platform/ydbops/pkg/maintenance"
)

func NewHostCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	maintenanceHostOpts := &options.MaintenanceHostOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "host",
		Short: "Request host from the CMS (Cluster Management System)",
		Long: `ydbops maintenance host: 
  Make a request to take the host out of the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			maintenanceHostOpts, rootOpts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskId, err := maintenance.RequestHost(*rootOpts, options.Logger, maintenanceHostOpts)
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

	maintenanceHostOpts.DefineFlags(cmd.PersistentFlags())
	options.RootOptionsInstance.DefineFlags(cmd.PersistentFlags())

	return cmd
}

func init() {
}
