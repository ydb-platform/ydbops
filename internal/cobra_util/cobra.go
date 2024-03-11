package cobra_util

import (
	"github.com/spf13/cobra"
	"github.com/ydb-platform/ydb-ops/pkg/options"
)

type PersistentPreRunEFunc func(cmd *cobra.Command, args []string) error

// Right now, Cobra does not support chaining PersistentPreRun commands.
// https://github.com/spf13/cobra/issues/216
//
// If we want to declare PersistentPreRun and also want parent's
// PersistentPreRun command called, we need to manually call it.
// This function is a wrapper that can be specified in PersistentPreRun
// commands of children, look at `ydb-ops restart storage` implementation.
func makePersistentPreRunE(original PersistentPreRunEFunc) PersistentPreRunEFunc {
	wrapped := func(cmd *cobra.Command, args []string) error {
		if parent := cmd.Parent(); parent != nil {
			if parent.PersistentPreRunE != nil {
				if err := parent.PersistentPreRunE(parent, args); err != nil {
					return err
				}
			}
		}

		return original(cmd, args)
	}

	return wrapped
}

func SetDefaultsOn(cmd *cobra.Command, opts options.Options) *cobra.Command {
	cmd.Flags().SortFlags = false
	cmd.PersistentFlags().SortFlags = false

	if cmd.PersistentPreRunE != nil {
		cmd.PersistentPreRunE = makePersistentPreRunE(
			func(cmd *cobra.Command, args []string) error {
				if opts != nil {
					return opts.Validate()
				}
        return nil
			},
		)
	}

	return cmd
}
