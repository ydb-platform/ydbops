package list

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

func New(f cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "list",
		Short: "List all existing maintenance tasks",
		Long: `ydbops maintenance list:
  List all existing maintenance tasks on the cluster.
  Can be useful if you lost your task id to refresh/complete your own task.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(),
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(f)
		},
	})

	return cmd
}
