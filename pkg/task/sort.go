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
	if l.Prog != nil && r.Prog == nil {
		return -1
	} else if l.Prog == nil && r.Prog != nil {
		return 1
	} else if l.Prog == nil && r.Prog == nil {
		return 2
	}
	if l.Prog.DoneCount > 0 && r.Prog.DoneCount == 0 {
		return -1
	} else if l.Prog.DoneCount == 0 && r.Prog.DoneCount > 0 {
		return 1
	}
	return 2
}

func sortCategory(l, r *Task) int {
	if l.Prog != nil && r.Prog == nil {
		return -1
	} else if l.Prog == nil && r.Prog != nil {
		return 1
	} else if l.Prog == nil && r.Prog == nil {
		return 2
	}
	if l.Prog.Category != "" && r.Prog.Category == "" {
		return -1
	} else if l.Prog.Category == "" && r.Prog.Category != "" {
		return 1
	} else if l.Prog.Category < r.Prog.Category {
		return -1
	} else if l.Prog.Category > r.Prog.Category {
		return 1
	}
	return 2
}

func sortPriority(l, r *Task) int {
	if l.Priority != nil && r.Priority == nil {
		return -1
	} else if l.Priority == nil && r.Priority != nil {
		return 1
	} else if l.Priority == nil && r.Priority == nil {
		return 2
	}
	if *l.Priority != "" && *r.Priority == "" {
		return -1
	} else if *l.Priority == "" && *r.Priority != "" {
		return 1
	} else if *l.Priority != "" && *r.Priority != "" && *l.Priority < *r.Priority {
		return -1
	} else if *l.Priority != "" && *r.Priority != "" && *l.Priority > *r.Priority {
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
	for ndx := len(tasks) - 1; ndx >= 0; ndx-- {
		if tasks[ndx].Parent != nil {
			tasks = slices.Delete(tasks, ndx, ndx+1)
		}
	}
	slices.SortFunc(tasks, sortHelper)
	var dfs func(int)
	dfs = func(ndx int) {
		children := slices.Clone(tasks[ndx].Children)
		slices.SortFunc(children, sortHelper)
		for sndx := len(children) - 1; sndx >= 0; sndx-- {
			tasks = slices.Insert(tasks, ndx+1, children[sndx])
			if len(children[sndx].Children) > 0 {
				dfs(ndx + 1)
			}
		}
	}
	for ndx := len(tasks) - 1; ndx >= 0; ndx-- {
		if len(tasks[ndx].Children) > 0 {
			dfs(ndx)
		}
	}
	return tasks
}
