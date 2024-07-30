package drop

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

func New(f cmdutil.Factory) *cobra.Command {
	taskIdOpts := &Options{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "drop",
		Short: "Drop an existing maintenance task",
		Long: `ydbops maintenance drop:
  Drops the maintenance task, meaning two things:
  1. Any hosts given within the maintenance task will be considered returned.
  2. Any hosts requested, but not yet given, will not be reserved for you any longer.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(), taskIdOpts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return f.GetCMSClient().DropTask(taskIdOpts.TaskID)
		},
	})

	taskIdOpts.DefineFlags(cmd.PersistentFlags())

	return cmd
}
