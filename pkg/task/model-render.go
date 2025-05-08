package task

import (
	"fmt"
	"strings"
)

// Used to carry intermediary info for a token,
// or a part of a token,
// or a formatted string from a token
type rToken struct {
	token         *Token
	raw           string
	color         string
	dominantColor string
}

// Used to carry intermediary info for a task
type rTask struct {
	tsk          *Task
	tokens       []*rToken // TODO: flatten
	id           int
	idColor      string
	idLen        int
	countLen     int // progress count
	doneCountLen int // progress doneCount
}

func (r *rTask) stringify(color bool, withIndex bool) string {
	var out strings.Builder

	if withIndex {
		idTxt := fmt.Sprintf("%0*d", r.idLen, r.id)
		if color {
			out.WriteString(colorize(r.idColor, idTxt))
		} else {
			out.WriteString(idTxt)
		}
		out.WriteRune(' ')
	}

	for _, tk := range r.tokens {
		if tk.color == "" {
			tk.color = "print.color-default"
		}
		if tk.token != nil && tk.token.Type == TokenProgress {
			parts := formatProgress(tk.token.Value.(*Progress), r.countLen, r.doneCountLen)
			for _, pt := range parts {
				if color {
					out.WriteString(colorizeToken(&pt))
				} else {
					out.WriteString(pt.raw)
				}
			}
			out.WriteRune(' ')
			continue
		}
		if color {
			out.WriteString(colorizeToken(tk))
		} else {
			out.WriteString(tk.raw)
		}
		out.WriteRune(' ')
	}
	return strings.TrimSpace(out.String())
}

// Used to carry intermediary info for a task List
type rList struct {
	tasks        []*rTask
	path         string
	maxLen       int
	idLen        int
	countLen     int          // progress count
	doneCountLen int          // progress doneCount
	idList       map[int]bool // a set of available ids
}

// Used to carry intermediary info for a printing session
type rPrint struct {
	lists        map[string]*rList
	maxLen       int
	idLen        int
	countLen     int // progress count
	doneCountLen int // progress doneCount
}
