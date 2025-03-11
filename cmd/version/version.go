package version

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ydb-platform/ydbops/pkg/command"
)

var ( // These variables are populated during build time using ldflags
	BuildTimestamp string
	BuildVersion   string
	BuildCommit    string
)

var VersionCommandDescription = command.NewDescription(
	"version",
	"Print ydbops version",
	"Print ydbops version and other build info: git commit and date",
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   VersionCommandDescription.GetUse(),
		Short: VersionCommandDescription.GetShortDescription(),
		Long:  VersionCommandDescription.GetLongDescription(),
		RunE: func(_ *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("free args not expected: %v", args)
			}
			fmt.Printf(
				"Git commit: %s\nTag: %s\nBuild date: %s\n",
				BuildCommit,
				BuildVersion,
				BuildTimestamp,
			)
			return nil
		},
	}

	return cmd
}
