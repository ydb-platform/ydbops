package maintenance

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/cmd/maintenance/complete"
	"github.com/ydb-platform/ydbops/cmd/maintenance/create"
	"github.com/ydb-platform/ydbops/cmd/maintenance/drop"
	"github.com/ydb-platform/ydbops/cmd/maintenance/list"
	"github.com/ydb-platform/ydbops/cmd/maintenance/refresh"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
)

type Options struct {
	*command.BaseOptions
}

func New(f cmdutil.Factory) *cobra.Command {
	options := &Options{}
	c := cli.SetDefaultsOn(&cobra.Command{
		Use:   "maintenance",
		Short: "Request hosts from the Cluster Management System",
		Long: `ydbops maintenance [command]:
    Manage host maintenance operations: request and return hosts
    with performed maintenance back to the cluster.`,
		PreRunE: cli.PopulateProfileDefaultsAndValidate(
			options.BaseOptions, options,
		),
		RunE: cli.RequireSubcommand,
	})

	c.AddCommand(
		complete.New(f),
		create.New(f),
		drop.New(f),
		list.New(f),
		refresh.New(f),
	)

	return c
}
