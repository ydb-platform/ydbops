package cluster

import (
	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/cmd/nodes/list"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
)

func New(f cmdutil.Factory) *cobra.Command {
	cmd := cli.SetDefaultsOn(&cobra.Command{
		Use:   "nodes",
		Short: "Node operations with the Cluster Management System",
		Long: `ydbops nodes [command]:
    Perform node operations.`,
		RunE: cli.RequireSubcommand,
	})

	cmd.AddCommand(
		list.New(f),
	)

	return cmd
}
