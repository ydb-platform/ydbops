package host

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
)

func NewHostCommand(f cmdutil.Factory) *cobra.Command {
	opts := &MaintenanceHostOpts{
		BaseOptions: &command.BaseOptions{},
	}
	return &cobra.Command{
		Use:   "host",
		Short: "Request host from the CMS (Cluster Management System)",
		Long: `ydbops maintenance host:
  Make a request to take the host out of the cluster.`,
		RunE: func(_ *cobra.Command, _ []string) error {
			taskId, err := maintenance.RequestHost(f.GetCMSClient(), &maintenance.RequestHostParams{
				AvailabilityMode:    opts.GetAvailabilityMode(),
				MaintenanceDuration: opts.GetMaintenanceDuration(),
				HostFQDN:            opts.HostFQDN,
			})
			if err != nil {
				return err
			}

			fmt.Printf(
				"Your task id is:\n\n%s\n\nPlease write it down for refreshing and completing the task later.\n",
				taskId,
			)

			return nil
		},
		PreRunE: cli.PopulateProfileDefaultsAndValidate(opts.BaseOptions, opts),
	}
}
