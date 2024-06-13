package command

import "github.com/spf13/cobra"

type Command interface {
	ToCobraCommand() *cobra.Command
	RegisterSubcommands(...Command)
}

type BaseCommandDescription struct {
	use              string
	shortDescription string
	longDescription  string
}

func NewDescription(use, shortDescription, longDescription string) *BaseCommandDescription {
	return &BaseCommandDescription{
		use:              use,
		shortDescription: shortDescription,
		longDescription:  longDescription,
	}
}

func (b *BaseCommandDescription) GetUse() string {
	return b.use
}

func (b *BaseCommandDescription) GetShortDescription() string {
	return b.shortDescription
}

func (b *BaseCommandDescription) GetLongDescription() string {
	return b.longDescription
}
