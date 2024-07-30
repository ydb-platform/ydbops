package create

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

func New(f cmdutil.Factory) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "create",
		Short: "Create a maintenance task to obtain a set of hosts",
		Long: `ydbops maintenance create:
  Create a maintenance task, which allows taking the set of hosts out of the cluster.`,
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
