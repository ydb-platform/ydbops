package cli

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydbops/internal/cli"
	"github.com/ydb-platform/ydbops/pkg/command"
	"github.com/ydb-platform/ydbops/pkg/options"
	"github.com/ydb-platform/ydbops/pkg/profile"
)

func PopulateProfileDefaultsAndValidate(rootOpts *command.BaseOptions, optsArgs ...options.Options) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		err := profile.FillDefaultsFromActiveProfile(rootOpts.ProfileFile, rootOpts.ActiveProfile)
		if err != nil {
			return err
		}

		for _, opts := range optsArgs {
			if err := opts.Validate(); err != nil {
				return fmt.Errorf("%w\nTry '--help' option for more info", err)
			}
		}
		return rootOpts.Validate()
	}
}

func RequireSubcommand(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("you have not selected a subcommand\nTry '--help' option for more info")
	}
	return nil
}

func SetDefaultsOn(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false

	cobra.AddTemplateFunc("drawNiceTree", func(cmd *cobra.Command) string {
		if cmd.HasAvailableSubCommands() {
			var builder strings.Builder
			builder.WriteString("Subcommands:")
			for _, line := range cli.GenerateCommandTree(cmd, 23) {
				builder.WriteString("\n")
				builder.WriteString(line)
			}
			builder.WriteString("\n")
			return builder.String()
		}
		return ""
	})

	cobra.AddTemplateFunc("generateUsage", cli.GenerateUsage)

	cobra.AddTemplateFunc("listAllFlagsInNiceGroups", func(cmd *cobra.Command) string {
		if cmd == cmd.Root() {
			return "Global options:\n" + cli.ColorizeUsages(cmd)
		}

		if cmd.HasAvailableSubCommands() {
			return strings.Join(cli.GenerateShortGlobalOptions(cmd.Root()), "\n")
		}

		if cmd.HasFlags() {
			return fmt.Sprintf(
				"Options:\n%s",
				strings.Join(cli.GenerateCommandOptionsMessage(cmd), "\n"),
			)
		}
		return ""
	})

	cmd.SetUsageTemplate(cli.UsageTemplate)

	return cmd
}
