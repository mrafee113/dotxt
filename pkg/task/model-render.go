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
	tokens       []*rToken
	id           int
	idColor      string
	idLen        int
	countLen     int // progress count
	doneCountLen int // progress doneCount
}

func (r *rTask) stringify(color bool, maxWidth int) string {
	var out strings.Builder
	var length int
	var fold func(string) string
	fold = func(str string) string {
		if maxWidth == -1 {
			return str
		}
		n := len(str)
		if n+length <= maxWidth {
			length += n
			return str
		}
		if n > maxWidth {
			oldLen := length
			length = r.idLen + 1
			return str[:maxWidth-oldLen-1] + "\\\n" +
				strings.Repeat(" ", r.idLen+1) +
				fold(str[maxWidth-oldLen-1:])
		}
		if r.idLen+1+len(str) > maxWidth {
			length = 0
			return "\n" + fold(strings.Repeat(" ", r.idLen+1)+str)
		}
		length = r.idLen + 1 + len(str)
		return "\n" + strings.Repeat(" ", r.idLen+1) + str
	}

	idTxt := fmt.Sprintf("%0*d", r.idLen, r.id)
	if color {
		out.WriteString(colorize(r.idColor, fold(idTxt)))
	} else {
		out.WriteString(fold(idTxt))
	}
	out.WriteString(fold(" "))

	for _, tk := range r.tokens {
		if tk.color == "" {
			tk.color = "print.color-default"
		}
		if tk.token != nil && tk.token.Type == TokenProgress {
			parts := formatProgress(tk.token.Value.(*Progress), r.countLen, r.doneCountLen)
			for _, pt := range parts {
				if color {
					out.WriteString(colorizeToken(fold(pt.raw), pt.color, pt.dominantColor))
				} else {
					out.WriteString(fold(pt.raw))
				}
			}
			out.WriteString(fold(" "))
			continue
		}
		if color {
			out.WriteString(colorizeToken(fold(tk.raw), tk.color, tk.dominantColor))
		} else {
			out.WriteString(fold(tk.raw))
		}
		out.WriteString(fold(" "))
	}
	return strings.TrimSpace(out.String())
}

// Used to carry intermediary info for a task List
type rList struct {
	tasks        []*rTask
	path         string
	maxLen       int
	idLen        int
	countLen     int             // progress count
	doneCountLen int             // progress doneCount
	idList       map[string]bool // a set of available ids
}

// Used to carry intermediary info for a printing session
type rPrint struct {
	lists        map[string]*rList
	maxLen       int
	idLen        int
	countLen     int // progress count
	doneCountLen int // progress doneCount
}
