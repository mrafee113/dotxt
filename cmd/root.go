package cmd

import (
	"fmt"
	"to-dotxt/config"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const version = "0.0.0"

var rootCmd = &cobra.Command{
	Use:          "To-DoTxt",
	Short:        fmt.Sprintf("To-DoTxt %s: a text based todo list inspired by todotxt", version),
	SilenceUsage: true,
}

func init() {
	rootCmd.SetHelpTemplate(`
{{ with (or .Long .Short) }}{{ . | trimTrailingWhitespaces }}

{{ end}}Usage:{{if .Runnable}}
  {{ .UseLine }} [flags]{{end}}{{if .HasAvailableSubCommands}}
  {{ .CommandPath }} [command]{{end}}{{if gt (len .Aliases) 0 }}

Aliases:
  {{ .NameAndAliases }}{{end}}{{if .HasExample}}

Examples:
{{ .Example }}{{end}}{{if .HasAvailableSubCommands}}}

Available Commands:
{{- range .Commands }}
  {{ rpad .NameAndAliases 20 }} {{ .Short }}
{{- end}}{{end}}{{if .HasAvailableFlags}}{{if not .Parent}}

Flags:
{{ .Flags.FlagUsages | trimTrailingWhitespaces }}{{else}}

{{ if .HasInheritedFlags }}Local {{end}}Flags:
{{ .LocalFlags.FlagUsages | trimTrailingWhitespaces }}{{if .HasInheritedFlags}}

Global Flags:
{{ .InheritedFlags.FlagUsages | trimTrailingWhitespaces }}{{end}}{{end}}{{end}}
`)
	cobra.OnInitialize(func() {
		arg, err := rootCmd.PersistentFlags().GetString("config")
		cobra.CheckErr(err)
		cobra.CheckErr(config.InitViper(arg))
	})
	rootCmd.PersistentFlags().StringP("config", "c", "", "yaml config filepath")
	rootCmd.PersistentFlags().Bool("color", false, "enable colored mode")
	viper.BindPFlag("color", rootCmd.PersistentFlags().Lookup("color"))
	rootCmd.PersistentFlags().Bool("conky", false, "enable conky mode")
	viper.BindPFlag("conky", rootCmd.PersistentFlags().Lookup("conky"))
	rootCmd.PersistentFlags().Bool("debug", false, "enable debugging mode")
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
}

func Execute() error {
	return rootCmd.Execute()
}
