package cmd

import (
	"to-dotxt/pkg/task"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(printCmd)
	setPrintCmdFlags()
}

var printCmd = &cobra.Command{
	Use:   "print [--from=<todolist=todo>]",
	Short: "print tasks from a list",
	Long: `print [--from=<todolist=todo>]
  print tasks from a list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		if err := task.LoadFile(path); err != nil {
			return err
		}
		if err := task.PrintTasks(path, 130); err != nil {
			return err
		}
		return nil
	},
}

func setPrintCmdFlags() {
	// printCmd.Flags().BoolP("color", "c", true, "print colorized output tailored for conky")
	printCmd.Flags().String("from", "", "designate the target todolist")
}
