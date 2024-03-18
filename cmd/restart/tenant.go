package restart

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/internal/cobra_util"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewTenantCmd() *cobra.Command {
	restartOpts := options.RestartOptionsInstance

	cmd := cobra_util.SetDefaultsOn(&cobra.Command{
		Use:   "tenant",
		Short: "Restarts a specified subset of tenant nodes",
		Long: `ydbops restart tenant:
  Restarts a specified subset of tenant nodes (also known as dynnodes). 
  Not specifying any filters will restart all tenant nodes`,
		RunE: cobra_util.RequireSubcommand,
	})

	restartOpts.TenantOptions.DefineFlags(cmd.Flags())

	return cmd
}

func init() {
}
