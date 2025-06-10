package task

import (
	"dotxt/config"
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

/* file structure

dotxt-config-dir/
	todos/
		todo
		_etc/
			todo.done
			todo.bak
		_archive/
			prev
			prev.done
			prev.bak

*/

const DefaultTodo = "todo"

func GetTodoPathArgFromCmd(cmd *cobra.Command, arg string) (string, error) {
	path, err := cmd.Flags().GetString(arg)
	if err != nil {
		return "", nil
	}
	if strings.TrimSpace(path) == "" {
		return DefaultTodo, nil
	}
	return path, nil
}

func mkDirs() error {
	cfgPath := config.ConfigPath()
	err := os.MkdirAll(filepath.Join(cfgPath, "todos", "_etc"), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	err = os.Mkdir(filepath.Join(cfgPath, "todos", "_archive"), 0755)
	if err != nil && !os.IsExist(err) {
		return err
	}
	return nil
}

func parseFilepath(path string) (string, error) {
	if path == "" {
		return filepath.Join(config.ConfigPath(), "todos", "todo"), nil
	}
	if filepath.IsAbs(path) {
		return path, nil
	}
	if tmpPath := filepath.Join(config.ConfigPath(), "todos", path); filepath.IsAbs(tmpPath) {
		return tmpPath, nil
	}
	return path, fmt.Errorf("%w: failed to parse filepath %s", terrors.ErrParse, path)
}

func prepFileTaskFromPath(path string) (string, error) {
	path, err := parseFilepath(path)
	if err != nil {
		return "", err
	}
	_, ok := FileTasks[path]
	if !ok {
		return "", fmt.Errorf("%w: %s", terrors.ErrListNotInMemory, path)
	}
	return path, nil
}

func CheckFileExistence(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	return fmt.Errorf("%w: %s", os.ErrNotExist, path)
}

func locateFiles() error {
	todoPath := filepath.Join(config.ConfigPath(), "todos")
	err := mkDirs()
	if err != nil {
		return err
	}
	files, err := os.ReadDir(todoPath)
	if err != nil {
		return fmt.Errorf("failed listing files from %s: %w", todoPath, err)
	}
	for _, entry := range files {
		if entry.Type().IsRegular() {
			key := filepath.Join(todoPath, entry.Name())
			_, ok := FileTasks[key]
			if !ok {
				FileTasks[key] = make([]*Task, 0)
			}
		}
	}
	return nil
}

func appendToDoneFile(text, path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if err = mkDirs(); err != nil {
		return err
	}
	path = strings.TrimPrefix(path, filepath.Join(config.ConfigPath(), "todos/"))
	path = filepath.Join(config.ConfigPath(), "todos", "_etc", path+".done")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0o655)
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	size := info.Size()
	if size > 0 {
		buf := make([]byte, 1)
		_, err = file.ReadAt(buf, size-1)
		if err != nil {
			return err
		}
		if buf[0] != '\n' {
			text = fmt.Sprintf("\n%s", text)
		}
	}
	_, err = file.Write([]byte(text))
	return err
}

func removeFromDoneFile(ids []int, path string) ([]string, error) {
	var tasks []string
	if len(ids) < 1 {
		return tasks, fmt.Errorf("%w: ids is empty", terrors.ErrValue)
	}
	path, err := parseFilepath(path)
	if err != nil {
		return tasks, err
	}
	if err = mkDirs(); err != nil {
		return tasks, err
	}
	path = strings.TrimPrefix(path, filepath.Join(config.ConfigPath(), "todos/"))
	path = filepath.Join(config.ConfigPath(), "todos", "_etc", path+".done")
	data, err := os.ReadFile(path)
	if err != nil {
		return tasks, err
	}
	lines := strings.Split(string(data), "\n")
	sort.Sort(sort.Reverse(sort.IntSlice(ids)))
	for _, id := range ids {
		if len(lines)-1 < id {
			return tasks, fmt.Errorf("%w: id '%d' exceeds number of lines in done file %s", terrors.ErrValue, id, path)
		}
		text := lines[id]
		lines = slices.Delete(lines, id, id+1)
		if validateEmptyText(text) == nil {
			tasks = append(tasks, text)
		}
	}
	err = os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
	if err != nil {
		return tasks, err
	}
	return tasks, nil
}

func CreateFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	if utils.FileExists(path) {
		return nil
	}
	if err = mkDirs(); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(""), 0644); err != nil {
		return err
	}
	locateFiles()
	return nil
}

func LoadFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	err = locateFiles()
	if err != nil {
		return err
	}
	if _, ok := FileTasks[path]; !ok || !utils.FileExists(path) {
		return os.ErrNotExist
	}
	fileTasks, err := ParseTasks(path)
	if err != nil {
		return err
	}
	FileTasks[path] = fileTasks
	return nil
}

func ReloadFiles() error {
	for path := range FileTasks {
		if err := LoadFile(path); err != nil {
			return err
		}
	}
	return nil
}

func LoadOrCreateFile(path string) error {
	err := LoadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if err != nil && os.IsNotExist(err) {
		err = CreateFile(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func StoreFile(path string) error {
	path, err := parseFilepath(path)
	if err != nil {
		return err
	}
	fileTasks, ok := FileTasks[path]
	if !ok {
		return fmt.Errorf("%w: %s", terrors.ErrListNotInMemory, path)
	}
	var lines []string
	for _, file := range fileTasks {
		var textArr []string
		for _, token := range file.Tokens {
			textArr = append(textArr, token.Raw)
		}
		lines = append(lines, strings.Join(textArr, " "))
	}
	if err = mkDirs(); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644); err != nil {
		return err
	}
	return nil
}

func StoreFiles() error {
	for path := range FileTasks {
		if err := StoreFile(path); err != nil {
			return err
		}
	}
	return nil
}

func taskifyRandomFile(path string) ([]Task, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return []Task{}, err
	}
	var tasks []Task
	for line := range strings.SplitSeq(string(data), "\n") {
		task, err := ParseTask(nil, line)
		if err != nil {
			continue
		}
		tasks = append(tasks, *task)
	}
	return tasks, nil
}

func LsFiles() ([]string, error) {
	var out []string
	if err := mkDirs(); err != nil {
		return out, err
	}
	rootDir := filepath.Join(config.ConfigPath(), "todos")
	rootDir, err := filepath.Abs(rootDir)
	if err != nil {
		return out, err
	}

	err = filepath.WalkDir(rootDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && strings.HasPrefix(d.Name(), "_") {
			return fs.SkipDir
		}
		if !d.IsDir() {
			out = append(out, path)
		}
		return nil
	})
	return out, err
}
