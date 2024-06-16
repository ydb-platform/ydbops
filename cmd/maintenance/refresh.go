package maintenance

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

func NewRefreshCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	taskIdOpts := &options.TaskIdOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "refresh",
		Short: "Try to obtain previously reserved hosts",
		Long: `ydbops maintenance refresh: 
  Performs a request to check whether any previously reserved hosts have become available.`,
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

			task, err := maintenance.RefreshTask(taskIdOpts)
			if err != nil {
				return err
			}

			fmt.Println(prettyprint.TaskToString(task))

			return nil
		},
	})

	taskIdOpts.DefineFlags(cmd.PersistentFlags())
	options.RootOptionsInstance.DefineFlags(cmd.PersistentFlags())

	return cmd
}

func init() {
}
