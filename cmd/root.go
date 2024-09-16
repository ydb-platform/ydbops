package cmd

import (
	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"

	"github.com/ydb-platform/ydbops/cmd/maintenance"
	"github.com/ydb-platform/ydbops/cmd/restart"
	"github.com/ydb-platform/ydbops/cmd/run"
	iCli "github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/cli"
	"github.com/ydb-platform/ydbops/pkg/cmdutil"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
)

var RootCommandDescription = command.NewDescription(
	"ydbops",
	"ydbops: a CLI tool for performing YDB cluster maintenance operations",
	"ydbops: a CLI tool for performing YDB cluster maintenance operations",
)

type RootOptions struct {
	*command.BaseOptions
}

func (r *RootOptions) Validate() error {
	return r.BaseOptions.Validate()
}

func (r *RootOptions) DefineFlags(fs *pflag.FlagSet) {
	r.BaseOptions.DefineFlags(fs)
}

func NewRootCommand(
	logLevelSetter zap.AtomicLevel,
	logger *zap.SugaredLogger,
	boptions *command.BaseOptions,
) *cobra.Command {
	roptions := &RootOptions{
		BaseOptions: boptions,
	}
	cmd := &cobra.Command{
		Use:   RootCommandDescription.GetUse(),
		Short: RootCommandDescription.GetShortDescription(),
		Long:  RootCommandDescription.GetLongDescription(),
		// hide --completion for more compact --help
		CompletionOptions: cobra.CompletionOptions{
			DisableDefaultCmd: true,
		},
		SilenceUsage: true,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			err := options.Validate()
			if err != nil {
				return err
			}

			logLevelSetter.SetLevel(roptions.VerbosityLevel.Level())

			zap.S().Debugf("Current logging level enabled: %s", roptions.VerbosityLevel.Level())
			return nil
		},
		RunE: cli.RequireSubcommand,
	}
	roptions.DefineFlags(cmd.PersistentFlags())

	cmd.SetHelpCommand(&cobra.Command{
		Hidden: true,
	})
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false
	cmd.SetOutput(color.Output)
	cmd.SetUsageTemplate(iCli.UsageTemplate)

	return cmd
}

func InitRootCommandTree(root *cobra.Command, f cmdutil.Factory) {
	root.AddCommand(
		restart.New(f),
		maintenance.New(f),
		run.New(f),
	)
}
