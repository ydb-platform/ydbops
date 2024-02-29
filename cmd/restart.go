package cmd

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/internal/util"
	"github.com/ydb-platform/ydb-ops/pkg/options"
)

func NewRestartCmd() *cobra.Command {
	opts := options.RestartOptionsInstance
	cmd := &cobra.Command{
		Use:   "restart",
		Short: "restart short description",
		Long:  `restart long description`,
		PersistentPreRunE: util.MakePersistentPreRunE(
			func(cmd *cobra.Command, args []string) error {
				return opts.Validate()
			},
		),
	}

	opts.DefineFlags(cmd.PersistentFlags())
	return cmd
}

func init() {
}
