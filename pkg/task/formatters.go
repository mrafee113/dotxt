package task

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"to-dotxt/pkg/utils"

	"github.com/spf13/viper"
)

var rightNow time.Time

func init() {
	rightNow = time.Now()
}

/*
	duration format rule of thumb

if 1.25 < year: 1.5y
if 1 < year < 1.25: 1y2m
if year < 1:

	if 2 < month: 3m
	if 1 < month < 2:
		if 1 < week: 1m2w
		if week < 1: 1m12d
	if month < 1:
		if 1 < week: 1w2d
		if week < 1:
			if 2 < day: 3d
			if 1 < day < 2: 1d2h
			if day < 1:
				if 2 < hour: 3h30M
				if hour < 2: 1h30M25S
*/
func formatDuration(d *time.Duration) string {
	totalSec := d.Seconds()
	var sign string
	if totalSec < 0 {
		sign = "-"
		totalSec = -1 * totalSec
	}

	const (
		secPerMin  = float64(60)
		secPerHr   = secPerMin * 60
		secPerDay  = secPerHr * 24
		secPerWeek = secPerDay * 7
		secPerMo   = secPerDay * 30
		secPerYr   = secPerDay * 365
	)

	calcRemaining := func(val, oldUnit, newUnit float64) float64 {
		return (totalSec - val*oldUnit) / newUnit
	}
	getIfPositive := func(format string, val float64) string {
		if val <= 0 || (strings.Contains(format, ".0f") && int(val) <= 0) {
			return ""
		}
		return fmt.Sprintf(format, val)
	}

	if totalSec == 0 {
		return "rn"
	}
	var dtStr string = func() string {
		if years := totalSec / secPerYr; 1.25 <= years {
			return fmt.Sprintf("%.2fy", years)
		}
		if years := totalSec / secPerYr; 1 <= years && years < 1.25 {
			return fmt.Sprintf("%.0fy", years) + getIfPositive("%.2fm", calcRemaining(years, secPerYr, secPerMo))
		}
		if months := totalSec / secPerMo; 2 <= months {
			return fmt.Sprintf("%.2fm", months)
		}
		if months := totalSec / secPerMo; 1 <= months && months < 2 {
			if weeks := calcRemaining(months, secPerMo, secPerWeek); 1 <= weeks {
				return fmt.Sprintf("%.0fm", months) + getIfPositive("%.2fw", weeks)
			}
			return fmt.Sprintf("%.0fm", months) + getIfPositive("%.0fd", calcRemaining(months, secPerMo, secPerDay))
		}
		if weeks := totalSec / secPerWeek; 1 <= weeks {
			return fmt.Sprintf("%.0fw", weeks) + getIfPositive("%.0fd", calcRemaining(weeks, secPerWeek, secPerDay))
		}
		if days := totalSec / secPerDay; 2 <= days {
			return fmt.Sprintf("%.0fd", days)
		}
		if days := totalSec / secPerDay; 1 <= days && days < 2 {
			return fmt.Sprintf("%0.fd", days) + getIfPositive("%.0f'", calcRemaining(days, secPerDay, secPerHr))
		}
		if hours := totalSec / secPerHr; 2 <= hours {
			return fmt.Sprintf("%0.f'", hours) + getIfPositive(`%.0f"`, calcRemaining(hours, secPerHr, secPerMin))
		}
		hours := totalSec / secPerHr
		mins := calcRemaining(hours, secPerHr, secPerMin)
		secs := totalSec - (mins * secPerMin)
		return fmt.Sprintf("%0.f'", hours) + getIfPositive(`%.0f"`, mins) + getIfPositive("%.0fs", secs)
	}()

	return sign + dtStr
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

func formatProgress(p *Progress) string {
	// TODO: progress color system
	percentage := 100 * p.Count / p.DoneCount
	barText := func(width int) string {
		pLen := width * p.Count / p.DoneCount
		switch pLen {
		case 0:
			return strings.Repeat(" ", width)
		case 1:
			return ">" + strings.Repeat(" ", width-1)
		case width:
			return strings.Repeat("=", width)
		}
		return strings.Repeat("=", pLen-1) + ">" + strings.Repeat(" ", width-pLen)
	}(10)

	return fmt.Sprintf(
		"%3d/%3d(%3d%%) %s (%s)",
		p.Count, p.DoneCount,
		percentage, barText, p.Unit,
	)
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

	add("print.color-index", strconv.Itoa(*t.ID))

	if t.Progress.DoneCount > 0 {
		addAsRegular(formatProgress(&t.Progress))
	}
	if t.Priority != "" {
		// TODO: priority color system
		addAsRegular(fmt.Sprintf("(%s)", t.Priority))
	}
	var reminderCount int

	for _, token := range t.Tokens {
		switch token.Type {
		case TokenPriority, TokenProgress:
			continue
		case TokenText:
			addAsRegular(token.Raw)
		case TokenID:
			add("print.color-"+token.Key, strconv.Itoa(token.Value.(int)))
		case TokenHint:
			var color string
			switch token.Key {
			case "@":
				color = "print.color-at"
			case "#":
				color = "print.color-tag"
			case "+":
				color = "print.color-plus"
			}
			add(color, token.Raw)
		case TokenDate:
			// TODO: make dynamic using config
			if slices.Contains([]string{"due", "end", "dead"}, token.Key) {
				val, err := t.Temporal.getField(token.Key)
				if err != nil {
					addAsRegular(token.Raw)
					continue
				}
				relStr, ok := temporalFormatFallback[token.Key]
				if !ok {
					addAsRegular(token.Raw)
					continue
				}
				rel, err := t.Temporal.getField(relStr)
				if err != nil {
					addAsRegular(token.Raw)
					continue
				}
				add("print.color-date"+token.Key, fmt.Sprintf("$%s=%s", token.Key, formatAbsoluteDatetime(val, rel)))
			}
			if strings.HasPrefix(token.Key, "r") {
				if reminderCount >= len(t.Temporal.Reminders) {
					addAsRegular(token.Raw)
					continue
				}
				val := t.Temporal.Reminders[reminderCount]
				relStr, ok := temporalFormatFallback["r"]
				if !ok {
					addAsRegular(token.Raw)
					continue
				}
				rel, err := t.Temporal.getField(relStr)
				if err != nil {
					addAsRegular(token.Raw)
					continue
				}
				add("print.color-date-r", fmt.Sprintf("$r=%s", formatAbsoluteDatetime(&val, rel)))
			}
		case TokenDuration:
			add("print.color-every", fmt.Sprintf("$every=%s", formatDuration(t.Every)))
		default:
			addAsRegular(token.Raw)
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
		taskText := task.Print(viper.GetBool("color"))
		maxLenTask = max(maxLenTask, len(taskText))
		out = append(out, taskText)
	}
	out = append([]string{fmt.Sprintf("> %s %s", filepath.Base(path), strings.Repeat("—", min(maxLenTask, maxLen)))}, out...)
	fmt.Println(strings.Join(out, "\n"))
	return nil
}
