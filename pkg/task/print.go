package task

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"to-dotxt/pkg/utils"
	"unicode"

	"github.com/spf13/viper"
)

var rightNow time.Time

func init() {
	rightNow = time.Now()
}

func formatDuration(d *time.Duration) string {
	totalSec := int(d.Seconds())
	var sign string
	if totalSec < 0 {
		sign = "-"
		totalSec = -1 * totalSec
	}

	const (
		secPerMin = 60
		secPerHr  = secPerMin * 60
		secPerDay = secPerHr * 24
		secPerMo  = secPerDay * 30
		secPerYr  = secPerDay * 365
	)

	years := totalSec / secPerYr
	totalSec %= secPerYr
	months := totalSec / secPerMo
	totalSec %= secPerMo
	days := totalSec / secPerDay
	totalSec %= secPerDay
	hours := totalSec / secPerHr
	totalSec %= secPerHr
	mins := totalSec / secPerMin
	secs := totalSec % secPerMin

	parts := []string{sign}
	if years > 0 {
		parts = append(parts, fmt.Sprintf("%dy", years))
	}
	if months > 0 {
		parts = append(parts, fmt.Sprintf("%02dm", months))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%02dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%02dh", hours))
	}
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%02dM", mins))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%02dS", secs))
	}

	return strings.Join(parts, "")
}

func formatAbsoluteDatetime(dt *time.Time, relDt *time.Time) string {
	if dt == nil {
		return ""
	}
	if relDt == nil {
		return dt.Format("2006-01-02T15-04")
	}
	d := dt.Sub(*relDt)
	return formatDuration(&d)
}

func (t *Task) Print(colorize bool) string {
	DebugTask(t)
	var out []string
	add := func(colorName, text string) {
		out = append(out, utils.Colorize(colorize, viper.GetString(colorName), text))
	}
	addAsRegular := func(text string) {
		out = append(out, text)
	}

	add("print.color-id", strconv.Itoa(*t.ID))

	line := *t.Text
	if len(line) > 0 && line[0] == '(' {
		j := strings.IndexRune(line, ')')
		if j > 0 && strings.IndexFunc(line[1:j], unicode.IsSpace) == -1 {
			// priority
			addAsRegular(line[0:j])
			line = line[j+1:]
		}
	}

	tokens := strings.Split(line, " ")
	for ndx := range tokens {
		token := strings.TrimSpace(tokens[ndx])
		if token == "" {
			continue
		}
		switch token[0] {
		case '+':
			add("print.color-plus", token)
		case '@':
			add("print.color-at", token)
		case '#':
			add("print.color-tag", token)
		case '$':
			keyValue := strings.SplitN(token[1:], "=", 2)
			if len(keyValue) != 2 {
				// regular text
				addAsRegular(token)
				continue
			}
			key, value := keyValue[0], keyValue[1]
			switch key {
			case "id": // EID
				add("print.color-eid", fmt.Sprintf("$id=%d", *t.EID))
			case "P": // PID
				add("print.color-pid", fmt.Sprintf("$pid=%d", *t.Parent))
			// case "c":
			// 	continue // creation date
			// case "lud":
			// 	continue // last updated date
			case "due": // due date
				add("print.color-due-date", fmt.Sprintf("$due=%s", formatAbsoluteDatetime(t.DueDate, &rightNow)))
			case "end": // end date
				add("print.color-end-date", fmt.Sprintf("$end=%s", formatAbsoluteDatetime(t.EndDate, t.DueDate)))
			case "dead": // deadline
				add("print.color-deadline", fmt.Sprintf("$dead=%s", formatAbsoluteDatetime(t.Deadline, t.DueDate)))
			case "every": // interval
				add("print.color-interval", fmt.Sprintf("$every=%s", formatDuration(t.Every)))
			case "r": // reminder
				reminder, err := parseAbsoluteDatetime(value)
				// if err != nil {
				// 	reminder, err = t.parseVariableDuration("due", value)
				// }
				if err != nil {
					// regular ass text
					addAsRegular(token)
					continue
				}
				add("print.color-reminder", fmt.Sprintf("$r=%s", formatAbsoluteDatetime(reminder, &rightNow)))
			case "p": // progress task
				var text []string
				if t.Unit != "" {
					text = append(text, fmt.Sprintf("(%s)", t.Unit))
				}
				text = append(text, fmt.Sprintf("%d/%d", t.Count, t.DoneCount))
				add("print.color-progress", fmt.Sprintf("$p=%s", strings.Join(text, " ")))
			default:
				addAsRegular(token)
			}
		default:
			addAsRegular(token)
		}
	}
	return strings.Join(out, " ")
}

func PrintTasks(path string, maxLen int) error {
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	var out []string
	var maxLenTask int = len(fmt.Sprintf("> %s", filepath.Base(path)))
	for _, task := range FileTasks[path] {
		taskText := task.Print(true)
		maxLenTask = max(maxLenTask, len(taskText))
		out = append(out, taskText)
	}
	out = append([]string{fmt.Sprintf("> %s %s", filepath.Base(path), strings.Repeat("—", min(maxLenTask, maxLen)))}, out...)
	fmt.Println(strings.Join(out, "\n"))
	return nil
}
