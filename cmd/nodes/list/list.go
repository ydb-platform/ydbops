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
		Short: "List the cluster nodes",
		Long: `ydbops cluster list:
  Obtain the list of cluster nodes from the Configuration Management System.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(), opts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			return opts.Run(f)
		},
	})

	opts.DefineFlags(cmd.PersistentFlags())

	return cmd
}
