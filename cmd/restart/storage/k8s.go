package storage

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters/storage_k8s"
)

func NewK8sCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	rootOpts := options.RootOptionsInstance

	cmd := &cobra.Command{
		Use:   "k8s",
		Short: "k8s short description",
		Long:  `k8s long description`,
		Run: func(cmd *cobra.Command, args []string) {
			rolling.PrepareRolling(restartOpts, rootOpts, options.Logger, &storage_k8s.Restarter{})
		},
	}

	return cmd
}

func init() {
}
