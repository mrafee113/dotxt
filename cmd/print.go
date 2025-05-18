package cmd

import (
	"to-dotxt/pkg/task"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func init() {
	rootCmd.AddCommand(printCmd)
	setPrintCmdFlags()
}

var printCmd = &cobra.Command{
	Use:   "print <todolist=todo>...",
	Short: "print tasks from lists",
	Long: `print <todolist=todo>...
  print tasks from lists`,
	RunE: func(cmd *cobra.Command, args []string) error {
		maxlen := viper.GetInt("maxlen")

		all, err := cmd.Flags().GetBool("all")
		if err != nil {
			return err
		}
		if len(args) < 1 {
			all = true
		}
		if all {
			paths, err := task.LsFiles()
			if err != nil {
				return err
			}
			for _, path := range paths {
				if err := task.LoadFile(path); err != nil {
					return err
				}
			}
			return task.PrintLists(paths, maxlen)
		}
		for _, arg := range args {
			if err := task.LoadFile(arg); err != nil {
				return err
			}
		}
		return task.PrintLists(args, maxlen)
	},
}

func setPrintCmdFlags() {
	printCmd.Flags().Bool("all", false, "print all lists")
	printCmd.Flags().Int("maxlen", 80, "maximum length")
	viper.BindPFlag("maxlen", printCmd.Flags().Lookup("maxlen"))
}
