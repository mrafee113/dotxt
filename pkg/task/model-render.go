package task

import (
	"fmt"
	"strings"
	"unicode"
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
	idPrefix := ""
	newLinePrefix := strings.Repeat(" ", r.idLen+1)
	if depth := r.tsk.Depth(); depth > 0 {
		depthSpace := strings.Repeat(" ", depth*(r.idLen+1))
		idPrefix += depthSpace
		newLinePrefix += depthSpace
	}
	var length int
	var fold func(string) string
	fold = func(str string) string {
		if maxWidth == -1 {
			return str
		}
		n := len(str)
		if n+length <= maxWidth { // fits
			length += n
			return str
		}
		if length >= maxWidth-1 { // current line has no space whatsoever
			length = r.idLen + 1
			if str == " " {
				return "\n" + newLinePrefix
			}
			return "\n" + newLinePrefix + fold(str)
		}
		if n > maxWidth || r.idLen+1+n > maxWidth { // string is so long it has to be split
			oldLen := length
			length = r.idLen + 1
			return str[:maxWidth-oldLen-1] + "\\\n" +
				newLinePrefix + fold(str[maxWidth-oldLen-1:])
		}
		// str is long enough to not fit current line and not long enough to be splitted
		length = r.idLen + 1 + n
		return "\n" + newLinePrefix + str
	}

	idTxt := fmt.Sprintf("%s%0*d", idPrefix, r.idLen, r.id)
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
	return strings.TrimRightFunc(out.String(), unicode.IsSpace)
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
