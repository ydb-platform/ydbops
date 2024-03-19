package cmd

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/options"
)

func NewRestartCmd() *cobra.Command {
	opts := options.RestartOptionsInstance

	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "restart",
		Short: "Restarts a specified subset of nodes in the cluster",
		Long: `ydbops restart: 
  Restarts a specified subset of nodes in the cluster. 
  Has subcommands for various YDB environments.`,
		RunE: cli.RequireSubcommand,
	})

	opts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
