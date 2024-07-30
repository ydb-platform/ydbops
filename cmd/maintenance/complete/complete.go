package complete

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/prettyprint"
)

func New(f cmdutil.Factory) *cobra.Command {
	opts := &Options{}

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "complete",
		Short: "Declare the maintenance task completed",
		Long: `ydbops maintenance complete:
  Any hosts that have been given to you within the task will be considered returned to the cluster.
  You must not perform any host maintenance after you called this command.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			f.GetBaseOptions(), opts,
		),
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := f.GetCMSClient().CompleteActions(opts.TaskID, opts.HostFQDNs)
			if err != nil {
				return err
			}

			fmt.Println(prettyprint.ResultToString(result))

			return nil
		},
	})

	opts.DefineFlags(cmd.PersistentFlags())

	return cmd
}
