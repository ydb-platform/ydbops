package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters/storage_baremetal"
)

func NewStorageCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := storage_baremetal.New()

	cmd := &cobra.Command{
		Use:   "storage",
		Short: "storage short description",
		Long:  `storage long description`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return restartOpts.Validate()
		},
		Run: func(cmd *cobra.Command, args []string) {
			rolling.PrepareRolling(restartOpts, rootOpts, options.Logger, restarter)
		},
	}

	restarter.Opts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
