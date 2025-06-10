package cmd

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"to-dotxt/pkg/task"
	"to-dotxt/pkg/terrors"
	"unicode"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(
		addCmd, delCmd, appendCmd,
		prependCmd, replaceCmd,
		deduplicateCmd, deprioritizeCmd,
		prioritizeCmd, doneCmd,
		revertCmd, moveCmd, migrateCmd,
		LsNCmd)
	setAddCmdFlags()
	setDelCmdFlags()
	setAppendCmdFlags()
	setPrependCmdFlags()
	setReplaceCmdFlags()
	setDeduplicateCmdFlags()
	setDeprioritizeCmdFlags()
	setPrioritizeCmdFlags()
	setRevertCmdFlags()
	setDoneCmdFlags()
	setMigrateCmdFlags()
	setLsNCmdFlags()
}

func loadFuncStoreFile(path string, f func() error) error {
	if err := task.LoadFile(path); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return task.StoreFile(path)
}

func loadorcreateFuncStoreFile(path string, f func() error) error {
	if err := task.LoadOrCreateFile(path); err != nil {
		return err
	}
	if err := f(); err != nil {
		return err
	}
	return task.StoreFile(path)
}

var addCmd = &cobra.Command{
	Use:   "add <task> [--to=<todolist=todo>]",
	Short: "add task",
	Long: `add <task> [--to=<todolist=todo>]
  adds task to todolist`,
	RunE: func(cmd *cobra.Command, args []string) error {
		arg := strings.Join(args, " ")
		path, err := task.GetTodoPathArgFromCmd(cmd, "to")
		if err != nil {
			return err
		}
		return loadorcreateFuncStoreFile(path, func() error {
			return task.AddTaskFromStr(arg, path)
		})
	},
}

func setAddCmdFlags() {
	addCmd.Flags().String("to", "", "designate the target todolist")
}

var delCmd = &cobra.Command{
	Use:   "del <id>... [--from=<todolist=todo>]",
	Short: "delete task",
	Long: `del|rm <id>... [--from=<todolist=todo>]
  removes task from todolist`,
	Aliases: []string{"rm"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("%w: failed to parse task id <%s>: %w", terrors.ErrParse, arg, err)
			}
			ids = append(ids, num)
		}

		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DeleteTasks(ids, path)
		})
	},
}

func setDelCmdFlags() {
	delCmd.Flags().String("from", "", "designate the target todolist")
}

var appendCmd = &cobra.Command{
	Use:   "app <id> <task> [--to=<todolist=todo>]",
	Short: "append to task",
	Long: `append|app <task> [--to=<todolist=todo>],
  appends text to the end of the designated task`,
	Aliases: []string{"append"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("%w: failed to parse id '%s': %w", terrors.ErrParse, args[0], err)
		}
		text := strings.Join(args[1:], " ")
		if strings.TrimSpace(text) == "" {
			return terrors.ErrEmptyText
		}

		path, err := task.GetTodoPathArgFromCmd(cmd, "to")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.AppendToTask(id, text, path)
		})
	},
}

func setAppendCmdFlags() {
	appendCmd.Flags().String("to", "", "designate the target todolist")
}

var prependCmd = &cobra.Command{
	Use:   "prepend <id> <task> [--to=<todolist=todo>]",
	Short: "prepend to task",
	Long: `prepend|prep <task> [--to=<todolist=todo>],
  prepends text to the end of the designated task`,
	Aliases: []string{"prep"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("%w: failed to parse id '%s': %w", terrors.ErrParse, args[0], err)
		}
		text := strings.Join(args[1:], " ")
		if strings.TrimSpace(text) == "" {
			return terrors.ErrEmptyText
		}

		path, err := task.GetTodoPathArgFromCmd(cmd, "to")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.PrependToTask(id, text, path)
		})
	},
}

func setPrependCmdFlags() {
	prependCmd.Flags().String("to", "", "designate the target todolist")
}

var replaceCmd = &cobra.Command{
	Use:   "replace <id> <task> [--to=<todolist=todo>]",
	Short: "replace line with a new task",
	Long: `replace|update <id> <task> [--to=<todolist=todo>]
  replace line with a new task`,
	Aliases: []string{"update"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		if len(args) < 2 {
			return terrors.ErrEmptyText
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := task.GetTodoPathArgFromCmd(cmd, "to")
		if err != nil {
			return err
		}
		text := strings.Join(args[1:], " ")
		return loadFuncStoreFile(path, func() error {
			return task.ReplaceTask(id, text, path)
		})
	},
}

func setReplaceCmdFlags() {
	replaceCmd.Flags().String("to", "", "designate the target todolist")
}

var deduplicateCmd = &cobra.Command{
	Use:   "deduplicate [--from=<todolist=todo>]",
	Short: "deduplicate list",
	Long: `deduplicate [--from=<todolist=todo>]
  removes duplicated lines from a list. repetetive spaces and meta variables are ignored.`,
	Aliases: []string{"dedup"},
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DeduplicateList(path)
		})
	},
}

func setDeduplicateCmdFlags() {
	deduplicateCmd.Flags().String("from", "", "designate the target todolist")
}

var deprioritizeCmd = &cobra.Command{
	Use:   "depri <id>... [--from=<todolist=todo>]",
	Short: "deprioritize task",
	Long: `depri|dp <id>... [--from=<todolist=todo>]
  deprioritizes task(s) (removes priority) from list`,
	Aliases: []string{"dp"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("%w: failed to parse task id <%s>: %w", terrors.ErrParse, arg, err)
			}
			ids = append(ids, num)
		}

		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			for _, id := range ids {
				if err = task.DeprioritizeTask(id, path); err != nil {
					return err
				}
			}
			return nil
		})
	},
}

func setDeprioritizeCmdFlags() {
	deprioritizeCmd.Flags().String("from", "", "designate the target todolist")
}

var prioritizeCmd = &cobra.Command{
	Use:   "pri <id> <priority> [--to=<todolist=todo>]",
	Short: "prioritize task",
	Long: `pri|p <id> <priority> [--to=<todolist=todo>]
  prioritizes task given the id and a priority text in a list`,
	Aliases: []string{"p"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		if len(args) < 2 {
			return terrors.NewArgNotProvidedError("priority")
		}
		if len(args) > 2 || strings.IndexFunc(args[1], unicode.IsSpace) != -1 {
			return fmt.Errorf("%w: priority cannot contain spaces", terrors.ErrValue)
		}

		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := task.GetTodoPathArgFromCmd(cmd, "to")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.PrioritizeTask(id, args[1], path)
		})
	},
}

func setPrioritizeCmdFlags() {
	prioritizeCmd.Flags().String("to", "", "designate the target todolist")
}

var doneCmd = &cobra.Command{
	Use:   "done <id> [--from=<todolist=todo>]",
	Short: "finish and move task",
	Long: `do|done <id> [--from=<todolist=todo>]
  finish task`,
	Aliases: []string{"do"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("id")
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("%w: failed to parse task id <%s>: %w", terrors.ErrParse, arg, err)
			}
			ids = append(ids, num)
		}
		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.DoneTask(ids, path)
		})
	},
}

func setDoneCmdFlags() {
	doneCmd.Flags().String("from", "", "designate the target todolist")
}

var revertCmd = &cobra.Command{
	Use:   "revert <id>... [--from=<todolist=todo>]",
	Short: "revert tasks from done to list",
	Long: `revert <id>... [--from=<todolist=todo>]
  reverts tasks from done to list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		var ids []int
		for _, arg := range args {
			num, err := strconv.Atoi(arg)
			if err != nil {
				return fmt.Errorf("%w: failed to parse task id <%s>: %w", terrors.ErrParse, arg, err)
			}
			ids = append(ids, num)
		}

		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}
		return loadFuncStoreFile(path, func() error {
			return task.RevertTask(ids, path)
		})
	},
}

func setRevertCmdFlags() {
	revertCmd.Flags().String("from", "", "designate the target todolist")
}

var moveCmd = &cobra.Command{
	Use:   "move <from> <id> <to>",
	Short: "move task around",
	Long: `move|mv <from> <id> <to>
  move task to another list`,
	Aliases: []string{"mv"},
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.NewArgNotProvidedError("from")
		}
		if len(args) < 2 {
			return terrors.NewArgNotProvidedError("id")
		}
		if len(args) < 3 {
			return terrors.NewArgNotProvidedError("to")
		}

		from, idString, to := args[0], args[1], args[2]
		if err := task.CheckFileExistence(from); err != nil {
			return err
		}
		id, err := strconv.Atoi(idString)
		if err != nil {
			return err
		}

		if err = task.LoadFile(from); err != nil {
			return err
		}
		if err = task.LoadOrCreateFile(to); err != nil {
			return err
		}
		if err = task.MoveTask(from, id, to); err != nil {
			return err
		}
		if err = task.StoreFile(to); err != nil {
			return err
		}
		if err = task.StoreFile(from); err != nil {
			return err
		}
		return nil
	},
}

var migrateCmd = &cobra.Command{
	Use:   "migrate <from> [--to=<todolist=todo>]",
	Short: "migrate tasks from a given file",
	Long: `migrate <from> [--to=<todolist=todo>]
  migrate tasks from a given file`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		from := args[0]
		to, err := cmd.Flags().GetString("to")
		if err != nil {
			return err
		}
		if strings.TrimSpace(to) == "" {
			to = filepath.Base(from)
		}
		return loadorcreateFuncStoreFile(to, func() error {
			return task.MigrateTasks(from, to)
		})
	},
}

func setMigrateCmdFlags() {
	migrateCmd.Flags().String("to", "", "designate the target todolist")
}

var LsNCmd = &cobra.Command{
	Use:   "lsn id [--from=<todolist=todo>]",
	Short: "print a single task from list",
	Long: `lsn id [--from=<todolist=todo>]
  print a single task from list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return terrors.ErrNoArgsProvided
		}
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return err
		}
		path, err := task.GetTodoPathArgFromCmd(cmd, "from")
		if err != nil {
			return err
		}

		if err := task.LoadFile(path); err != nil {
			return err
		}
		return task.PrintTask(id, path)
	},
}

func setLsNCmdFlags() {
	LsNCmd.Flags().String("from", "", "designate the target todolist")
}
