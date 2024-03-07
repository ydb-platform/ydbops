package storage

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

func NewBaremetalCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewStorageBaremetalRestarter()

	cmd := &cobra.Command{
		Use:   "baremetal",
		Short: "TODO baremetal short description",
		Long:  `TODO baremetal long description`,
		PersistentPreRunE: util.MakePersistentPreRunE(
			func(cmd *cobra.Command, args []string) error {
				return restarter.Opts.Validate()
			},
		),
		Run: func(cmd *cobra.Command, args []string) {
			rolling.PrepareRolling(restartOpts, rootOpts, options.Logger, restarter)
		},
	}

	restarter.Opts.DefineFlags(cmd.Flags())
	return cmd
}

func init() {
}
