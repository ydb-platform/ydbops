package cli

import (
	"fmt"
	"slices"
	"strings"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func DeterminePadding(curCommand, subCommandLineNumber, totalCommands int) string {
	if curCommand == totalCommands-1 {
		if subCommandLineNumber == 0 {
			return "└─ "
		}
		return "   "
	}

	if subCommandLineNumber == 0 {
		return "├─ "
	}
	return "│  "
}

func GenerateUsage(cmd *cobra.Command) string {
	boldUsage := color.New(color.Bold).Sprint("Usage:")
	if cmd == cmd.Root() {
		return fmt.Sprintf("%s ydbops [global options...] <subcommand>", boldUsage)
	}

	cmdChain := cmd.Name()

	curCmd := cmd
	for curCmd.HasParent() && curCmd.Parent() != curCmd.Root() {
		cmdChain = curCmd.Parent().Name() + " " + cmdChain
		curCmd = curCmd.Parent()
	}

	subcommand := ""
	if cmd.HasAvailableSubCommands() {
		subcommand = "<subcommand>"
	}

	return fmt.Sprintf("%s ydbops [global options...] %s [options] %s",
		boldUsage,
		cmdChain,
		subcommand,
	)
}

func GenerateCommandTree(cmd *cobra.Command, paddingSize int) []string {
	bold := color.New(color.Bold)

	result := []string{bold.Sprint(cmd.Name()) + strings.Repeat(" ", paddingSize-len(cmd.Name())) + cmd.Short}
	if cmd.HasAvailableSubCommands() {
		subCommandLen := len(cmd.Commands())
		for i := 0; i < len(cmd.Commands()); i++ {
			subCmd := cmd.Commands()[i]
			if !subCmd.Hidden {
				subCmdTree := GenerateCommandTree(subCmd, paddingSize-3)
				for j, line := range subCmdTree {
					result = append(result, DeterminePadding(i, j, subCommandLen)+line)
				}
			}
		}
	}
	return result
}

func GenerateShortGlobalOptions(rootCmd *cobra.Command) []string {
	flagNames := []string{}
	rootCmd.Flags().VisitAll(func(f *pflag.Flag) {
		var flagString string
		greenName := color.GreenString("--" + f.Name)
		if len(f.Shorthand) > 0 {
			greenShorthand := color.GreenString("-" + f.Shorthand)
			flagString = fmt.Sprintf("{%s|%s}", greenShorthand, greenName)
		} else {
			flagString = greenName
		}

		flagNames = append(flagNames, flagString)
	})

	return []string{
		"Global options: ",
		"  " + strings.Join(flagNames, ", "),
		"  To get full description of these options run 'ydbops --help'.",
	}
}

func ColorizeUsages(cmd *cobra.Command) string {
	flagOccurences := []string{}
	cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
		longFlagName := fmt.Sprintf("--%s", f.Name)
		flagOccurences = append(flagOccurences, longFlagName)
		if len(f.Shorthand) > 0 {
			shortFlagName := fmt.Sprintf("-%s", f.Shorthand)
			flagOccurences = append(flagOccurences, shortFlagName)
		}
	})

	// Imagine flags --profile and --profile-name. If we first recolor
	// --profile, then --profile-name would be recolored into <green>--profile-name</green>.
	// Sorting in descending order by length allows to avoid that.
	slices.SortFunc(flagOccurences, func(a string, b string) int {
		return len(b) - len(a)
	})

	replacementPairs := []string{}

	for _, flag := range flagOccurences {
		replacementPairs = append(replacementPairs, flag, color.GreenString(flag))
	}

	replacer := strings.NewReplacer(replacementPairs...)

	flagUsages := cmd.LocalFlags().FlagUsages()

	return replacer.Replace(flagUsages)
}

func GenerateCommandOptionsMessage(cmd *cobra.Command) []string {
	result := []string{}

	local := cmd.LocalFlags()
	if len(local.FlagUsages()) > 0 {
		if cmd == cmd.Root() {
			return GenerateShortGlobalOptions(cmd)
		}

		result = append(result, ColorizeUsages(cmd))
	}

	if cmd.HasParent() {
		result = append(result, GenerateCommandOptionsMessage(cmd.Parent())...)
	}

	return result
}
