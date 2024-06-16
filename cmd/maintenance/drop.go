package maintenance

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewDropCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	taskIdOpts := &options.TaskIdOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "drop",
		Short: "Drop an existing maintenance task",
		Long: `ydbops maintenance drop: 
  Drops the maintenance task, meaning two things:
  1. Any hosts given within the maintenance task will be considered returned.
  2. Any hosts requested, but not yet given, will not be reserved for you any longer.`,
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

			err = maintenance.DropTask(taskIdOpts)
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
