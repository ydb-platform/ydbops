package util

import (
	"github.com/spf13/cobra"
)

type PersistentPreRunEFunc func(cmd *cobra.Command, args []string) error

// Right now, Cobra does not support chaining PersistentPreRun commands.
// https://github.com/spf13/cobra/issues/216
//
// If we want to declare PersistentPreRun and also want parent's
// PersistentPreRun command called, we need to manually call it.
// This function is a wrapper that can be specified in PersistentPreRun
// commands of children, look at `ydb-ops restart storage` implementation.
func MakePersistentPreRunE(original PersistentPreRunEFunc) PersistentPreRunEFunc {
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
