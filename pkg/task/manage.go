package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"slices"
	"sort"
	"strings"
)

func cleanupIDs(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	n := Lists.Len(path)
	ndxs := make(map[int]bool)
	for _, task := range Lists[path].Tasks {
		if task.ID != nil {
			if _, ok := ndxs[*task.ID]; *task.ID >= n || *task.ID < 0 || ok {
				task.ID = nil
			}
		}
		if task.ID != nil {
			ndxs[*task.ID] = true
		}
	}
	var stack []int
	for ndx := range n {
		_, ok := ndxs[ndx]
		if !ok {
			stack = append(stack, ndx)
		}
	}
	for _, task := range Lists[path].Tasks {
		if task.ID == nil {
			task.ID = utils.MkPtr(stack[0])
			stack = stack[1:]
		}
	}

	idSortFunc := func(l, r *Task) int {
		if *l.ID < *r.ID {
			return -1
		} else if *l.ID > *r.ID {
			return 1
		} else {
			return 0
		}
	}
	Lists.SortFunc(path, idSortFunc)
	return nil
}

func cleanupRelations(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	Lists[path].EIDs = make(map[string]*Task)
	Lists[path].PIDs = make(map[*Task]string)
	for _, task := range Lists[path].Tasks {
		if task.EID != nil {
			Lists[path].EIDs[*task.EID] = task
		}
		if task.PID != nil {
			Lists[path].PIDs[task] = *task.PID
		}
		task.Children = make([]*Task, 0)
		task.Parent = nil
	}
	for task, pid := range Lists[path].PIDs {
		parent, ok := Lists[path].EIDs[pid]
		if !ok {
			continue
		}
		task.Parent = parent
		parent.Children = append(parent.Children, task)
	}
	var dfs func(*Task)
	dfs = func(node *Task) {
		for _, child := range node.Children {
			child.EIDCollapse = node.EIDCollapse
			dfs(child)
		}
	}
	for _, task := range Lists[path].Tasks {
		dfs(task)
	}
	return nil
}

func AddTask(task *Task, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	Lists.Append(path, task)
	cleanupIDs(path)
	cleanupRelations(path)
	return nil
}

func AddTaskFromStr(task, path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	t, err := ParseTask(nil, task)
	if err != nil {
		return err
	}
	return AddTask(t, path)
}

func getTaskIndexFromId(id int, path string) (int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return -1, err
	}
	taskNdx := -1
	for ndx, t := range Lists[path].Tasks {
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
	return Lists[path].Tasks[taskNdx], nil
}

func AppendToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	err = task.updateFromText(task.Raw() + " " + text)
	if err != nil {
		return err
	}
	task.ID = &id
	cleanupRelations(path)
	return nil
}

func PrependToTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	var newText string
	if task.Priority != nil && *task.Priority != "" {
		var out []string
		task.Tokens.Filter(func(tk *Token) bool {
			return tk.Type != TokenPriority
		}).ForEach(func(tk *Token) {
			out = append(out, tk.String(task))
		})
		curText := strings.Join(out, " ")
		newText = fmt.Sprintf("(%s) %s %s", *task.Priority, text, curText)
	} else {
		newText = text + " " + task.Raw()
	}
	err = task.updateFromText(newText)
	if err != nil {
		return err
	}
	task.ID = &id
	cleanupRelations(path)
	return nil
}

func ReplaceTask(id int, text, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	err = task.updateFromText(text)
	if err != nil {
		return err
	}
	task.ID = &id
	cleanupRelations(path)
	return nil
}

func DeduplicateList(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	var indexes []int
	var lines map[string][]int = make(map[string][]int)
	for ndx, task := range Lists[path].Tasks {
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
	if (task.Priority != nil && *task.Priority == "") || task.Priority == nil {
		return nil
	}

	for ndx := len(task.Tokens) - 1; ndx >= 0; ndx-- {
		if task.Tokens[ndx].Type == TokenPriority {
			task.Tokens = slices.Delete(task.Tokens, ndx, ndx+1)
			break
		}
	}
	task.Priority = nil
	return nil
}

func PrioritizeTask(id int, priority, path string) error {
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}

	if len(priority) > 2 && priority[0] == '(' && priority[len(priority)-1] == ')' {
		priority = priority[1 : len(priority)-1]
	}
	hadPriority := task.Priority != nil && *task.Priority != ""
	task.Priority = &priority
	pToken := Token{
		Type: TokenPriority, raw: fmt.Sprintf("(%s)", priority), Key: "priority",
		Value: utils.MkPtr(strings.TrimSuffix(strings.TrimPrefix(priority, "("), ")")),
	}
	if hadPriority {
		for ndx := range task.Tokens {
			if task.Tokens[ndx].Type == TokenPriority {
				task.Tokens[ndx] = &pToken
			}
		}
	} else {
		task.Tokens = append([]*Token{&pToken}, task.Tokens...)
	}
	return nil
}

// The indexes are sorted in decreasing order
func getIndexesFromIds(ids []int, path string) ([]int, error) {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return []int{}, err
	}
	if len(ids) == 0 {
		return []int{}, nil
	}

	var idsMap map[int]bool = make(map[int]bool)
	for _, val := range ids {
		idsMap[val] = true
	}
	var indexes []int
	for ndx, task := range Lists[path].Tasks {
		if task.ID == nil {
			continue
		}
		if _, ok := idsMap[*task.ID]; ok {
			indexes = append(indexes, ndx)
		}
	}
	sort.Sort(sort.Reverse(sort.IntSlice(indexes)))
	if len(indexes) != len(idsMap) {
		var ndxMap map[int]bool = make(map[int]bool)
		for _, val := range indexes {
			ndxMap[val] = true
		}
		var notFound []string
		for ndx := range idsMap {
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
		Lists.DeleteTasks(path, ndx, ndx+1)
	}
	err = cleanupIDs(path)
	if err != nil {
		return err
	}
	cleanupRelations(path)
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
		tasks = append(tasks, Lists[path].Tasks[ndx])
		Lists.DeleteTasks(path, ndx, ndx+1)
	}
	cleanupIDs(path)
	cleanupRelations(path)

	var out []string
	for _, task := range tasks {
		out = append(out, task.Raw())
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

	Lists.Append(to, Lists[from].Tasks[taskNdx])
	cleanupIDs(to)
	cleanupRelations(to)
	Lists.DeleteTasks(from, taskNdx, taskNdx+1)
	cleanupIDs(from)
	cleanupRelations(from)
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
		newId := Lists.Len(path)
		task, err := ParseTask(&newId, text)
		if err != nil {
			return err
		}
		Lists.Append(path, task)
	}
	cleanupRelations(path)
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
	if task.Prog == nil {
		return fmt.Errorf("%w: task '%d' does not have a progress associated with it", terrors.ErrValue, id)
	}
	rVal := task.Prog.Count + value
	task.Prog.Count = max(min(rVal, task.Prog.DoneCount), 0)
	pToken, _ := task.Tokens.Find(TkByType(TokenProgress))
	if pToken == nil {
		return fmt.Errorf("%w: task '%d' does not have a progress associated with it", terrors.ErrValue, id)
	}
	prog := task.Prog
	progText, err := unparseProgress(*prog)
	if err != nil {
		return err
	}
	progText = fmt.Sprintf("$p=%s", progText)
	pToken.raw = progText
	pToken.Value = utils.MkPtr(*prog)
	return nil
}

func CheckAndRecurTasks(path string) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	for _, task := range Lists[path].Tasks {
		if task.Time.Every != nil &&
			task.Time.DueDate != nil &&
			task.Time.DueDate.Before(rightNow) {

			newDt := *task.Time.DueDate
			for newDt.Before(rightNow) {
				newDt = newDt.Add(*task.Time.Every)
			}
			diff := newDt.Sub(*task.Time.DueDate) // must be before update!
			err := task.updateDate("due", &newDt)
			if err != nil {
				return err
			}

			if task.Time.EndDate != nil && task.Time.EndDate.Before(rightNow) {
				err = task.updateDate("end", utils.MkPtr(task.Time.EndDate.Add(diff)))
				if err != nil {
					return err
				}
			} else if task.Time.Deadline != nil && task.Time.Deadline.Before(rightNow) {
				err = task.updateDate("dead", utils.MkPtr(task.Time.Deadline.Add(diff)))
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}
