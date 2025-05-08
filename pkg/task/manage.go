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
	if task.ID == nil {
		id := len(FileTasks[path]) - 1
		task.ID = &id
	}
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
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return nil, err
	}
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

	var newText string
	if task.Priority != "" {
		newText = (*task.Text)[strings.IndexRune(*task.Text, ')')+1:]
		newText = fmt.Sprintf("(%s) ", task.Priority) + text + " " + newText
	} else {
		newText = text + " " + *task.Text
	}
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

	dummy, err := ParseTask(nil, text)
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
		taskNorm := task.Norm()
		_, ok := lines[taskNorm]
		if !ok {
			lines[taskNorm] = []int{ndx}
		} else {
			lines[taskNorm] = append(lines[taskNorm], ndx)
		}
	}
	for _, bucket := range lines {
		if len(bucket) > 1 {
			indexes = append(indexes, bucket[1:]...)
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
	for ndx := len(task.Tokens) - 1; ndx >= 0; ndx-- {
		if task.Tokens[ndx].Type == TokenPriority {
			task.Tokens = slices.Delete(task.Tokens, ndx, ndx+1)
			break
		}
	}
	task.Priority = ""
	return nil
}

func PrioritizeTask(id int, priority, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	if !strings.HasPrefix(priority, "(") || !strings.HasSuffix(priority, ")") {
		priority = fmt.Sprintf("(%s)", priority)
	}
	hadPriority := task.Priority != ""
	task.Priority = priority
	*task.Text = task.Priority + " " + *task.Text
	task.PText = task.Priority + " " + task.PText
	pToken := Token{
		Type: TokenPriority, Raw: priority, Key: "priority",
		Value: strings.TrimSuffix(strings.TrimPrefix(priority, "("), ")"),
	}
	if hadPriority {
		for ndx := range task.Tokens {
			if task.Tokens[ndx].Type == TokenPriority {
				task.Tokens[ndx] = pToken
			}
		}
	} else {
		task.Tokens = append([]Token{pToken}, task.Tokens...)
	}
	return nil
}

// The indexes are sorted in decreasing order
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
	var eids []int
	for _, ndx := range indexes {
		if eid := FileTasks[path][ndx].EID; eid != nil {
			eids = append(eids, *eid)
		}
		FileTasks[path] = slices.Delete(FileTasks[path], ndx, ndx+1)
	}
	// remove orphans
	for ndx := len(FileTasks[path]) - 1; ndx >= 0; ndx-- {
		if p := FileTasks[path][ndx].Parent; p != nil && slices.Contains(eids, *p) {
			FileTasks[path] = slices.Delete(FileTasks[path], ndx, ndx+1)
		}
	}
	return nil
}

func DoneTask(ids []int, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
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
		var textArr []string
		for _, token := range task.Tokens {
			textArr = append(textArr, token.Raw)
		}
		out = append(out, strings.Join(textArr, " "))
	}
	return appendToDoneFile(strings.Join(out, "\n"), path)
}

func MoveTask(from string, id int, to string) error {
	taskNdx, err := getTaskIndexFromId(id, from)
	if err != nil {
		return err
	}
	from, err = prepFileTaskFromPath(from)
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

func RevertTask(ids []int, path string) error {
	texts, err := removeFromDoneFile(ids, path)
	if err != nil {
		return err
	}
	path, err = prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	for _, text := range texts {
		newId := len(FileTasks[path])
		task, err := ParseTask(&newId, text)
		if err != nil {
			return err
		}
		FileTasks[path] = append(FileTasks[path], task)
	}
	return nil
}

func MigrateTasks(from, to string) error {
	tasks, err := taskifyRandomFile(from)
	if err != nil {
		return err
	}
	for _, t := range tasks {
		if err := AddTask(&t, to); err != nil {
			return err
		}
	}
	return nil
}

func IncrementProgressCount(id int, path string, value int) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	if task.Progress.DoneCount == 0 {
		return fmt.Errorf("%w: task '%d' does not have a progress associated with it", terrors.ErrValue, id)
	}
	task.Progress.Count = min(task.Progress.Count+value, task.Progress.DoneCount)
	var pToken *Token
	for ndx := range task.Tokens {
		if task.Tokens[ndx].Type == TokenProgress {
			pToken = &task.Tokens[ndx]
		}
	}
	if pToken == nil {
		return fmt.Errorf("%w: task '%d' does not have a progress associated with it", terrors.ErrValue, id)
	}
	prevRaw := pToken.Raw
	prog := task.Progress
	progText, err := unparseProgress(prog)
	if err != nil {
		return err
	}
	progText = fmt.Sprintf("$p=%s", progText)
	*task.Text = strings.Replace(*task.Text, prevRaw, progText, 1)
	task.PText = strings.Replace(task.PText, prevRaw, progText, 1)
	pToken.Raw = progText
	return nil
}
