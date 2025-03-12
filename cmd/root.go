package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/ydb-platform/ydbops/cmd/maintenance"
	nodes "github.com/ydb-platform/ydbops/cmd/nodes"
	"github.com/ydb-platform/ydbops/cmd/restart"
	"github.com/ydb-platform/ydbops/cmd/run"
	"github.com/ydb-platform/ydbops/cmd/version"
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
		Long:  fmt.Sprintf("%s (%s)", RootCommandDescription.GetLongDescription(), version.BuildVersion),
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

			logLevel := "info"
			if roptions.Verbose {
				logLevel = "debug"
			}

			lvc, err := zapcore.ParseLevel(logLevel)
			if err != nil {
				logger.Warn("Failed to set level")
				return err
			}
			logLevelSetter.SetLevel(lvc)

			zap.S().Debugf("Current logging level enabled: %s", logLevel)
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
		nodes.New(f),
		maintenance.New(f),
		run.New(f),
		version.New(),
	)
}
