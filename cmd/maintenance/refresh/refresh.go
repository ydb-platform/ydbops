package refresh

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

func New(f cmdutil.Factory) *cobra.Command {
	taskIdOpts := &Options{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "refresh",
		Short: "Try to obtain previously reserved hosts",
		Long: `ydbops maintenance refresh:
  Performs a request to check whether any previously reserved hosts have become available.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(), taskIdOpts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, err := f.GetCMSClient().RefreshTask(taskIdOpts.TaskID)
			if err != nil {
				return err
			}

			fmt.Println(prettyprint.TaskToString(task))

			return nil
		},
	})

	taskIdOpts.DefineFlags(cmd.PersistentFlags())

	return cmd
}
