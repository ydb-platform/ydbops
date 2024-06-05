package maintenance

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewCompleteCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	taskIdOpts := &options.TaskIdOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "complete",
		Short: "Declare the maintenance task completed",
		Long: `ydbops maintenance complete: 
  Any hosts that have been given to you within the task will be considered returned to the cluster.
  You must not perform any host maintenance after you called this command.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			taskIdOpts, rootOpts,
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

			err = maintenance.CompleteTask(taskIdOpts)

			if err != nil {
				return err
			}
			return nil
		},
	})

	taskIdOpts.DefineFlags(cmd.PersistentFlags())
	options.RootOptionsInstance.DefineFlags(cmd.PersistentFlags())

	return cmd
}

func init() {
}
