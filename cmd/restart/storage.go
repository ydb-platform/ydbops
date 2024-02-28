package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/pkg/options"
	"github.com/ydb-platform/ydb-ops/pkg/rolling"
	"github.com/ydb-platform/ydb-ops/pkg/rolling/restarters/storage_baremetal"
)

func NewStorageCmd() *cobra.Command {
	opts := options.RestartOptionsInstance

	cmd := &cobra.Command{
		Use:   "storage",
		Short: "storage short description",
		Long:  `storage long description`,
		Run: func(cmd *cobra.Command, args []string) {
			rolling.PrepareRolling(opts, options.Logger, &storage_baremetal.Restarter{})
		},
	}

	return cmd
}

func init() {
}
