package cobra_util

const UsageTemplate = `{{ generateUsage . }}{{if .HasExample}}

Examples:
  {{.Example}}{{end}} 

{{ drawNiceTree . }}
{{ listAllFlagsInNiceGroups . }}{{if .HasHelpSubCommands}}

Additional help topics:{{range .Commands}}{{if .IsAdditionalHelpTopicCommand}}
  {{rpad .CommandPath .CommandPathPadding}} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
