package cmd

import (
	"dotxt/pkg/task"
	"dotxt/pkg/terrors"
	"strconv"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(printCmd, toggleCollapseCmd)
	setPrintCmdFlags()
	setToggleCollapsedCmdFlags()
}

var printCmd = &cobra.Command{
	Use:   "print <todolist=todo>...",
	Short: "print tasks from lists",
	Long: `print <todolist=todo>...
  print tasks from lists`,
	RunE: func(cmd *cobra.Command, args []string) error {
		maxlen, err := cmd.Flags().GetInt("maxlen")
		if err != nil {
			return err
		}
		minlen, err := cmd.Flags().GetInt("minlen")
		if err != nil {
			return err
		}

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
			return task.PrintLists(paths, maxlen, minlen)
		}
		for _, arg := range args {
			if err := task.LoadFile(arg); err != nil {
				return err
			}
		}
		return task.PrintLists(args, maxlen, minlen)
	},
}

func setPrintCmdFlags() {
	printCmd.Flags().Bool("all", false, "print all lists")
	printCmd.Flags().Int("maxlen", 80, "maximum length")
	printCmd.Flags().Int("minlen", 80, "maximum length")
}

var toggleCollapseCmd = &cobra.Command{
	Use:   "tc id [--list==<todolist=todo>]",
	Short: "toggle the collapse/expanse of the children of a task",
	Long: `tc id [--list==<todolist=todo>]
  toggle the collapse/expanse of the children of a task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := prepTodoListArg(cmd)
		if err != nil {
			return err
		}

		return loadFuncStoreFile(path, func() error {
			return task.ToggleCollapsed(id, path)
		})
	},
}

func setToggleCollapsedCmdFlags() {
	toggleCollapseCmd.Flags().String("list", "", "designate the target todolist")
}
