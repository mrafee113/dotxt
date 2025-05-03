package task

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"to-dotxt/pkg/terrors"
)

func AddTask(task *Task, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	FileTasks[path] = append(FileTasks[path], task)
	return nil
}

func getTaskIndexFromId(id int, path string) (int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return -1, err
	}
	taskNdx := -1
	for ndx, t := range FileTasks[path] {
		if *t.ID == id {
			taskNdx = ndx
		}
	}
	if taskNdx == -1 {
		return taskNdx, fmt.Errorf("%w: task corresponding to id %d not found", terrors.ErrNotFound, id)
	}
	return taskNdx, nil
}

func getTaskFromId(id int, path string) (*Task, error) {
	taskNdx, err := getTaskIndexFromId(id, path)
	if err != nil {
		return nil, err
	}
	return FileTasks[path][taskNdx], nil
}

func AppendToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	newText := *task.Text + " " + text
	dummy, err := ParseTask(nil, newText)
	if err != nil {
		return fmt.Errorf("failed creating dummy task with updated text: %w", err)
	}
	*task = *dummy
	task.ID = &id
	return nil
}

func PrependToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	newText := text + " " + *task.Text
	dummy, err := ParseTask(nil, newText)
	if err != nil {
		return fmt.Errorf("failed creating dummy task with updated text: %w", err)
	}
	*task = *dummy
	task.ID = &id
	return nil
}

func ReplaceTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	newText := text
	dummy, err := ParseTask(nil, newText)
	if err != nil {
		return fmt.Errorf("failed creating dummy task with updated text: %w", err)
	}
	*task = *dummy
	task.ID = &id
	return nil
}

func DeduplicateList(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	var indexes []int
	var lines map[string][]int = make(map[string][]int)
	for ndx, task := range FileTasks[path] {
		_, ok := lines[task.PText]
		if !ok {
			lines[task.PText] = []int{ndx}
		} else {
			lines[task.PText] = append(lines[task.PText], ndx)
		}
	}
	for _, bucket := range lines {
		if len(bucket) > 1 {
			indexes = append(indexes, bucket...)
		}
	}
	return DeleteTasks(indexes, path)
}

func DeprioritizeTask(id int, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Priority == "" {
		return nil
	}

	pString := fmt.Sprintf("(%s)", task.Priority)
	task.PText = strings.TrimPrefix(task.PText, pString)
	task.PText = strings.TrimPrefix(task.PText, " ")
	*task.Text = strings.TrimPrefix(*task.Text, pString)
	*task.Text = strings.TrimPrefix(*task.Text, " ")
	task.Priority = ""
	return nil
}

func PrioritizeTask(id int, priority, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Priority != "" {
		return fmt.Errorf("task already has priority")
	}

	task.Priority = fmt.Sprintf("(%s)", priority)
	*task.Text = task.Priority + " " + *task.Text
	task.PText = task.Priority + " " + task.PText
	return nil
}

func getIndexesFromIds(ids []int, path string) ([]int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return []int{}, err
	}

	var idsMap map[int]bool = make(map[int]bool)
	for _, val := range ids {
		idsMap[val] = true
	}
	var indexes []int
	for ndx, task := range FileTasks[path] {
		if task.ID == nil {
			continue
		}
		if _, ok := idsMap[*task.ID]; ok {
			indexes = append(indexes, ndx)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indexes)))
	if len(indexes) != len(ids) {
		var ndxMap map[int]bool = make(map[int]bool)
		for _, val := range indexes {
			ndxMap[val] = true
		}
		var notFound []string
		for _, ndx := range ids {
			if _, ok := ndxMap[ndx]; !ok {
				notFound = append(notFound, fmt.Sprintf("%d", ndx))
			}
		}
		return []int{}, fmt.Errorf("%w: ids not found: %s", terrors.ErrNotFound, strings.Join(notFound, ", "))
	}
	return indexes, nil
}

func DeleteTasks(ids []int, path string) error {
	indexes, err := getIndexesFromIds(ids, path)
	if err != nil {
		return err
	}
	for _, ndx := range indexes {
		FileTasks[path] = slices.Delete(FileTasks[path], ndx, ndx+1)
	}
	return nil
}

func DoneTask(ids []int, path string) error {
	indexes, err := getIndexesFromIds(ids, path)
	if err != nil {
		return err
	}
	var tasks []*Task
	for _, ndx := range indexes {
		tasks = append(tasks, FileTasks[path][ndx])
		FileTasks[path] = slices.Delete(FileTasks[path], ndx, ndx+1)
	}

	var out []string
	for _, task := range tasks {
		out = append(out, *task.Text)
	}
	return appendToDoneFile(strings.Join(out, "\n"), path)
}

func MoveTask(from string, id int, to string) error {
	taskNdx, err := getTaskIndexFromId(id, from)
	if err != nil {
		return err
	}
	to, err = prepFileTaskFromPath(to)
	if err != nil {
		return err
	}

	FileTasks[to] = append(FileTasks[to], FileTasks[from][taskNdx])
	FileTasks[from] = slices.Delete(FileTasks[from], taskNdx, taskNdx+1)
	return nil
}

func RevertTask(id int, path string) error {
	text, err := removeFromDoneFile(id, path)
	if err != nil {
		return err
	}
	path, err = prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	newId := len(FileTasks[path])
	task, err := ParseTask(&newId, text)
	if err != nil {
		return err
	}
	FileTasks[path] = append(FileTasks[path], task)
	return nil
}
