package maintenance

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/client"
	"github.com/ydb-platform/ydbops/pkg/maintenance"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewGetCmd() *cobra.Command {
	rootOpts := options.RootOptionsInstance

	taskIdOpts := &options.TaskIdOpts{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "get",
		Short: "Get status of maintenance task",
		Long: `ydbops maintenance get: 
  Describe the maintenance task given its id.`,
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

			err = maintenance.GetTask(taskIdOpts)

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
