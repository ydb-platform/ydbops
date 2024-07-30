package restart

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/rolling"
)

var RestartCommandDescription = command.NewDescription(
	"restart",
	"Restarts a specified subset of nodes in the cluster",
	`ydbops restart:
  Restarts a specified subset of nodes in the cluster.
  By default will restart all nodes, filters can be specified to
  narrow down what subset will be restarted.`)

func New(
	f cmdutil.Factory,
) *cobra.Command {
	opts := &Options{
		RestartOptions: &rolling.RestartOptions{},
	}
	cmd := &cobra.Command{
		Use:     RestartCommandDescription.GetUse(),
		Short:   RestartCommandDescription.GetShortDescription(),
		Long:    RestartCommandDescription.GetLongDescription(),
		PreRunE: cli.PopulateProfileDefaultsAndValidate(f.GetBaseOptions(), opts.RestartOptions),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("Free args not expected: %v", args)
			}

			err := opts.Validate()
			if err != nil {
				return err
			}

			return opts.Run(f)
		},
	}
	opts.DefineFlags(cmd.Flags())

	return cmd
}
