package cmd

import (
	"strconv"
	"to-dotxt/pkg/task"
	"to-dotxt/pkg/terrors"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(incCmd)
	setIncCmdFlags()
}

var incCmd = &cobra.Command{
	Use:   "inc id [val=1] [--from=<todolist=todo>]",
	Short: "increment the count of a progress task",
	Long: `inc id [val=1] [--from=<todolist=todo>]
  increment the count of a progress task`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		val := 1
		if len(args) >= 2 {
			val, err = strconv.Atoi(args[1])
			if err != nil {
				return err
			}
		}
		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}

		return loadFuncStoreFile(path, func() error {
			return task.IncrementProgressCount(id, path, val)
		})
	},
}

func setIncCmdFlags() {
	incCmd.Flags().String("from", "", "designate the target todolist")
}
