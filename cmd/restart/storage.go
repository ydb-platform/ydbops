package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
)

func NewStorageCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance
	cmd := &cobra.Command{
		Use:   "storage",
		Short: "storage short description",
		Long:  `storage long description`,
		PersistentPreRunE: util.MakePersistentPreRunE(
			func(cmd *cobra.Command, args []string) error {
				return restartOpts.Validate()
			},
		),
	}
	return cmd
}

func init() {
}
