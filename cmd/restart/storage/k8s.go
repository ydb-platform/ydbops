package storage

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters"
)

func NewK8sCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance
	restarter := restarters.NewStorageK8sRestarter()

	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "k8s short description",
		Long:  `k8s long description`,
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
