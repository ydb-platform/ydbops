package run

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

var RunCommandDescription = command.NewDescription(
	"run",
	"Run an arbitrary executable (e.g. shell code) in the context of the local machine",
	`ydbops restart run:
		Run an arbitrary executable (e.g. shell code) in the context of the local machine
		(where rolling-restart is launched). For example, if you want to execute ssh commands
		on the ydb cluster node, you must write ssh commands yourself. See the examples.

		For every node released by CMS, ydbops will execute this payload independently.

		Restart will be treated as successful if your executable finished with a zero
		return code.

		Certain environment variable will be passed to your executable on each run:
			$HOSTNAME: the fqdn of the node currently released by CMS.`,
)

func New(
	f cmdutil.Factory,
) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}
	cmd := &cobra.Command{
		Use:     RunCommandDescription.GetUse(),
		Short:   RunCommandDescription.GetShortDescription(),
		Long:    RunCommandDescription.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(f.GetBaseOptions(), opts),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}
			return opts.Run(f)
		},
	}

	opts.DefineFlags(cmd.Flags())
	return cmd
}
