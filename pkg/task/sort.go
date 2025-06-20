package task

import (
	"slices"
	"strings"
)

func sortNil(l, r *Task) int {
	if l == nil && r != nil {
		return 1
	} else if l != nil && r == nil {
		return -1
	} else if l == nil && r == nil {
		return 0
	}
	return 2
}

func sortDoneCount(l, r *Task) int {
	if l.DoneCount > 0 && r.DoneCount == 0 {
		return -1
	} else if l.DoneCount == 0 && r.DoneCount > 0 {
		return 1
	}
	return 2
}

func sortCategory(l, r *Task) int {
	if l.Category != "" && r.Category == "" {
		return -1
	} else if l.Category == "" && r.Category != "" {
		return 1
	} else if l.Category < r.Category {
		return -1
	} else if l.Category > r.Category {
		return 1
	}
	return 2
}

func sortPriority(l, r *Task) int {
	if l.Priority != "" && r.Priority == "" {
		return -1
	} else if l.Priority == "" && r.Priority != "" {
		return 1
	} else if l.Priority != "" && r.Priority != "" && l.Priority < r.Priority {
		return -1
	} else if l.Priority != "" && r.Priority != "" && l.Priority > r.Priority {
		return 1
	}
	return 2
}

func sortHints(l, r *Task) int {
	extractPlus := func(hints []*string) []string {
		var out []string
		for _, h := range hints {
			if len(*h) > 0 && (*h)[0] == '+' {
				out = append(out, *h)
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
	return 2
}

func sortText(l, r *Task) int {
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
	return 2
}

func sortHelper(l, r *Task) int {
	if v := sortNil(l, r); v != 2 {
		return v
	} else if v = sortDoneCount(l, r); v != 2 {
		return v
	} else if v = sortCategory(l, r); v != 2 {
		return v
	} else if v = sortPriority(l, r); v != 2 {
		return v
	} else if v = sortHints(l, r); v != 2 {
		return v
	} else if v = sortText(l, r); v != 2 {
		return v
	}
	return 0
}

func sortTasks(tasks []*Task) []*Task {
	parentsToChildren := func() map[*Task][]*Task {
		parents := make(map[*Task][]*Task)
		parentIds := make(map[int]*Task)
		for _, task := range tasks {
			if task == nil {
				continue
			}
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
		if task == nil {
			continue
		}
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
