package task

import (
	"slices"
	"strings"
)

func sortHelper(l, r *Task) int {
	if l == nil && r != nil {
		return 1
	} else if l != nil && r == nil {
		return -1
	} else if l == nil && r == nil {
		return 0
	} else if l.DoneCount > 0 && r.DoneCount == 0 {
		return -1
	} else if l.DoneCount == 0 && r.DoneCount > 0 {
		return 1
	} else if l.DoneCount > 0 && r.DoneCount > 0 && l.Category < r.Category {
		return -1
	} else if l.DoneCount > 0 && r.DoneCount > 0 && l.Category > r.Category {
		return 1
	} else if l.Priority != "" && r.Priority == "" {
		return -1
	} else if l.Priority == "" && r.Priority != "" {
		return 1
	} else if l.Priority != "" && r.Priority != "" && l.Priority < r.Priority {
		return -1
	} else if l.Priority != "" && r.Priority != "" && l.Priority > r.Priority {
		return 1
	}
	extractPlus := func(hints []string) []string {
		var out []string
		for _, h := range hints {
			if len(h) > 0 && h[0] == '+' {
				out = append(out, h)
			}
		}
		slices.Sort(out)
		return out
	}
	lHints, rHints := extractPlus(l.Hints), extractPlus(r.Hints)
	lHint, rHint := strings.Join(lHints, ","), strings.Join(rHints, ",")
	if len(lHints) > 0 && len(rHints) == 0 {
		return -1
	} else if len(lHints) == 0 && len(rHints) > 0 {
		return 1
	} else if len(lHints) > 0 && len(rHints) > 0 && lHint < rHint {
		return -1
	} else if len(lHints) > 0 && len(rHints) > 0 && lHint > rHint {
		return 1
	}
	lText, rText := l.NormRegular(), r.NormRegular()
	if len(lText) > 0 && len(rText) == 0 {
		return -1
	} else if len(lText) == 0 && len(rText) > 0 {
		return 1
	} else if len(lText) > 0 && len(rText) > 0 && lText < rText {
		return -1
	} else if len(lText) > 0 && len(rText) > 0 && lText > rText {
		return 1
	}
	return 0
}

func sortTasks(tasks []*Task) []*Task {
	parentsToChildren := func() map[*Task][]*Task {
		parents := make(map[*Task][]*Task)
		parentIds := make(map[int]*Task)
		for _, task := range tasks {
			if task.EID != nil {
				parentIds[*task.EID] = task
				parents[task] = make([]*Task, 0)
			}
		}
		for ndx := len(tasks) - 1; ndx >= 0; ndx-- {
			if tasks[ndx].Parent != nil {
				parent, ok := parentIds[*tasks[ndx].Parent]
				if !ok {
					continue
				}
				parents[parent] = append(parents[parent], tasks[ndx])
				tasks = slices.Delete(tasks, ndx, ndx+1)
			}
		}
		return parents
	}()
	slices.SortFunc(tasks, sortHelper)

	var out []*Task
	for _, task := range tasks {
		out = append(out, task)
		children, ok := parentsToChildren[task]
		if !ok {
			continue
		}
		slices.SortFunc(children, sortHelper)
		out = append(out, children...)
	}
	return out
}
