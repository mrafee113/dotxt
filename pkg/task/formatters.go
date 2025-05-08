package task

import (
	"fmt"
	"maps"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"to-dotxt/pkg/terrors"
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

func formatPriorities(tasks []*rTask) error {
	if len(tasks) == 0 {
		return nil
	}

	colorSaturation := viper.GetFloat64("print.priority.saturation")
	colorLightness := viper.GetFloat64("print.priority.lightness")
	maxDepth := viper.GetInt("print.priority.group-depth")
	colorStartHue := viper.GetFloat64("print.priority.start-hue")
	colorEndHue := viper.GetFloat64("print.priority.end-hue")

	// AI generated...
	var assignHue func([]string, float64, float64, int) []string
	assignHue = func(tasks []string, startHue, endHue float64, depth int) []string {
		n := len(tasks)
		// base case: single task or max depth
		if n == 1 || depth >= maxDepth {
			hexes := make([]string, n)
			for i := range tasks {
				// evenly space within [startHue, endHue]
				h := startHue + (float64(i)+0.5)/float64(n)*(endHue-startHue)
				hexes[i] = utils.HslToHex(h, colorSaturation, colorLightness)
			}
			return hexes
		}

		// group by the next character of prefix
		groups := make([][]string, 0, n)
		prefixes := make([]string, 0, n)
		for _, t := range tasks {
			p := ""
			if len(t) > depth {
				p = t[:depth+1]
			}
			if len(groups) == 0 || prefixes[len(prefixes)-1] != p {
				groups = append(groups, []string{t})
				prefixes = append(prefixes, p)
			} else {
				groups[len(groups)-1] = append(groups[len(groups)-1], t)
			}
		}

		// if only one group formed, stop recursing deeper
		if len(groups) == 1 {
			return assignHue(tasks, startHue, endHue, maxDepth)
		}

		// otherwise recurse on each group, slicing the hue span
		out := make([]string, 0, n)
		span := (endHue - startHue) / float64(len(groups))
		for i, grp := range groups {
			h0 := startHue + float64(i)*span
			h1 := h0 + span
			out = append(out, assignHue(grp, h0, h1, depth+1)...)
		}
		return out
	}

	var priorities []string
	reverseMap := make(map[string][]int)
	for ndx, t := range tasks {
		if t.tsk != nil && t.tsk.Priority != "" {
			priorities = append(priorities, t.tsk.Priority)
			reverseMap[t.tsk.Priority] = append(reverseMap[t.tsk.Priority], ndx)
		}
	}
	slices.Sort(priorities)
	pColors := assignHue(priorities, colorStartHue, colorEndHue, 0)
	for cNdx := range pColors {
		for _, ndx := range reverseMap[priorities[cNdx]] {
			for _, tk := range tasks[ndx].tokens {
				if tk.token != nil && tk.token.Type == TokenPriority {
					tk.color = pColors[cNdx]
				}
			}
		}
	}
	return nil
}

func formatProgress(p *Progress, countLen, doneCountLen int) []rToken {
	colorizePercentage := func(percent int) string {
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
		// 0-120 red to green TODO: make dynamic
		hue := 120.0 * float64(percent) / 100.0
		saturation := viper.GetFloat64("print.progress.percentage.saturation")
		lightness := viper.GetFloat64("print.progress.percentage.lightness")
		return utils.HslToHex(hue, saturation, lightness)
	}

	percentage := 100 * p.Count / p.DoneCount
	percentageColor := colorizePercentage(percentage)
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
	}(viper.GetInt("print.progress.bartext-len"))

	return []rToken{
		{raw: fmt.Sprintf("%*d", countLen, p.Count), color: "print.progress.count"},
		{raw: fmt.Sprintf("/%*d", doneCountLen, p.DoneCount), color: "print.progress.done-count"},
		{raw: fmt.Sprintf("(%3d%%)", percentage), color: percentageColor},
		{raw: fmt.Sprintf(" %s ", barText), color: percentageColor},
		{raw: fmt.Sprintf("(%s)", p.Unit), color: "print.progress.unit"},
	}
}

func formatListHeader(list *rList) string {
	var out strings.Builder
	out.WriteString("> ")
	out.WriteString(filepath.Base(list.path))
	out.WriteString(" | ") // TODO: add reports
	out.WriteString(strings.Repeat("—", list.maxLen-out.Len()))
	out.WriteRune('\n')
	return colorize("print.color-header", out.String())
}

func resolvColor(color string) string {
	if strings.TrimSpace(color) == "" {
		color = "print.color-default"
	}
	if validateHexColor(color) != nil {
		color = viper.GetString(color)
	}
	return color
}

func colorize(color, text string) string {
	return utils.Colorize(resolvColor(color), text)
}

func colorizeToken(tk *rToken) string {
	if tk.dominantColor != "" {
		return colorize(tk.dominantColor, tk.raw)
	}
	return colorize(tk.color, tk.raw)
}

func colorizeIds(ids map[int]bool) map[int]string {
	out := make(map[int]string)
	if len(ids) == 0 {
		return out
	}
	startHue := viper.GetFloat64("print.ids.start-hue")
	endHue := viper.GetFloat64("print.ids.end-hue")
	saturation := viper.GetFloat64("print.ids.saturation")
	lightness := viper.GetFloat64("print.ids.lightness")
	ndx := 0
	n := float64(len(ids))
	for _, id := range slices.Sorted(maps.Keys(ids)) {
		h := startHue + float64(ndx)*(endHue-startHue)/n
		out[id] = utils.HslToHex(h, saturation, lightness)
		ndx++
	}
	return out
}

func formatID(tk Token) string {
	return fmt.Sprintf("$%s=%d", tk.Key, tk.Value.(int))
}

func formatCategoryHeader(category string, list *rList) string {
	var out strings.Builder
	out.WriteString(strings.Repeat(" ", list.idLen+1))
	out.WriteString(fmt.Sprintf("+ %s ", category))
	out.WriteString(strings.Repeat("—", list.maxLen-out.Len()))
	out.WriteRune('\n')
	return colorize("print.progress.header", out.String())
}

func (t *Task) Render(listMetadata *rList) *rTask {
	if listMetadata == nil {
		listMetadata = &rList{}
	}
	out := rTask{tsk: t, id: *t.ID, idColor: "print.color-index"}
	addAsRegular := func(token *Token) {
		out.tokens = append(out.tokens, &rToken{token: token, raw: token.Raw})
	}
	specialTokenMap := func() map[string]*Token {
		out := make(map[string]*Token)
		for ndx := range t.Tokens {
			if slices.Contains([]TokenType{
				TokenDate, TokenID, TokenDuration,
				TokenPriority, TokenProgress,
			}, t.Tokens[ndx].Type) {
				out[t.Tokens[ndx].Key] = &t.Tokens[ndx]
			}
		}
		return out
	}()

	if t.Progress.DoneCount > 0 {
		out.tokens = append(out.tokens, &rToken{token: specialTokenMap["p"]})
		listMetadata.countLen = max(listMetadata.countLen, len(strconv.Itoa(t.Progress.Count)))
		listMetadata.doneCountLen = max(listMetadata.doneCountLen, len(strconv.Itoa(t.Progress.DoneCount)))
	}
	if t.Priority != "" {
		out.tokens = append(out.tokens, &rToken{
			token: specialTokenMap["priority"],
			raw:   fmt.Sprintf("(%s)", t.Priority),
		})
	}

	var dominantColor, defaultColor string = func() (string, string) {
		if t.Temporal.DueDate != nil && t.Temporal.DueDate.Sub(rightNow) <= 0 {
			if t.Temporal.EndDate == nil && t.Temporal.Deadline == nil {
				return "print.color-burnt", ""
			}
			if t.Temporal.EndDate != nil {
				if t.Temporal.EndDate.Sub(rightNow) <= 0 {
					return "print.color-burnt", ""
				}
				return "", "print.color-running-event-text"
			}
			if t.Temporal.Deadline != nil {
				if t.Temporal.Deadline.Sub(rightNow) <= 0 {
					return "print.color-burnt", ""
				}
				return "", ""
			}
		}
		return "", ""
	}()

	var reminderCount int
	for ndx := range t.Tokens {
		switch t.Tokens[ndx].Type {
		case TokenPriority, TokenProgress:
			continue
		case TokenText:
			out.tokens = append(out.tokens, &rToken{
				token: &t.Tokens[ndx], raw: t.Tokens[ndx].Raw,
			})
		case TokenID:
			out.tokens = append(out.tokens, &rToken{
				token: &t.Tokens[ndx],
				raw:   formatID(t.Tokens[ndx]),
				color: "",
			})
			listMetadata.idList[t.Tokens[ndx].Value.(int)] = true
		case TokenHint:
			var color string
			switch t.Tokens[ndx].Key {
			case "@":
				color = "print.color-at"
			case "#":
				color = "print.color-tag"
			case "+":
				color = "print.color-plus"
			}
			out.tokens = append(out.tokens, &rToken{
				token: &t.Tokens[ndx],
				raw:   t.Tokens[ndx].Raw, color: color,
			})
		case TokenDate:
			if slices.Contains([]string{"due", "end", "dead"}, t.Tokens[ndx].Key) {
				val, err := t.Temporal.getField(t.Tokens[ndx].Key)
				if err != nil {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				relStr, ok := temporalFormatFallback[t.Tokens[ndx].Key]
				if !ok {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				rel, err := t.Temporal.getField(relStr)
				if err != nil {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				color := "print.color-date-" + t.Tokens[ndx].Key
				if t.Temporal.DueDate != nil && t.Temporal.DueDate.Sub(rightNow) <= 0 {
					if t.Tokens[ndx].Key == "due" {
						color = "print.color-burnt"
					}
					if t.Temporal.Deadline != nil && t.Temporal.Deadline.Sub(rightNow) > 0 &&
						t.Tokens[ndx].Key == "dead" {
						color = "print.color-imminent-deadline"
					}
					if t.Temporal.EndDate != nil && t.Temporal.EndDate.Sub(rightNow) > 0 &&
						t.Tokens[ndx].Key == "end" {
						color = "print.color-running-event"
					}
				}
				out.tokens = append(out.tokens, &rToken{
					token: &t.Tokens[ndx],
					raw:   fmt.Sprintf("$%s=%s", t.Tokens[ndx].Key, formatAbsoluteDatetime(val, rel)),
					color: color,
				})
			}
			if strings.HasPrefix(t.Tokens[ndx].Key, "r") {
				if reminderCount >= len(t.Temporal.Reminders) {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				val := t.Temporal.Reminders[reminderCount]
				relStr, ok := temporalFormatFallback["r"]
				if !ok {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				rel, err := t.Temporal.getField(relStr)
				if err != nil {
					addAsRegular(&t.Tokens[ndx])
					continue
				}
				if val.Sub(rightNow) < 0 {
					continue
				}
				out.tokens = append(out.tokens, &rToken{
					token: &t.Tokens[ndx],
					raw:   fmt.Sprintf("$r=%s", formatAbsoluteDatetime(&val, rel)),
					color: "print.color-date-r",
				})
			}
		case TokenDuration:
			out.tokens = append(out.tokens, &rToken{
				token: &t.Tokens[ndx],
				raw:   fmt.Sprintf("$every=%s", formatDuration(t.Every)),
				color: "print.color-every",
			})
		default:
			addAsRegular(&t.Tokens[ndx])
		}
	}
	listMetadata.maxLen = max(listMetadata.maxLen, len(out.stringify(false, true)))
	listMetadata.idLen = max(listMetadata.idLen, len(strconv.Itoa(*t.ID)))
	if dominantColor != "" || defaultColor != "" {
		for _, rtk := range out.tokens {
			rtk.dominantColor = dominantColor
			if rtk.token.Type == TokenText && defaultColor != "" {
				rtk.color = defaultColor
			}
		}
	}
	return &out
}

func RenderList(sessionMetadata *rPrint, path string) error {
	if sessionMetadata == nil {
		return fmt.Errorf("%w: session metadata must not be nil", terrors.ErrValue)
	}
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}

	FileTasks[path] = sortTasks(FileTasks[path])
	listMetadata := rList{path: path, idList: make(map[int]bool)}
	sessionMetadata.lists[path] = &listMetadata
	for _, task := range FileTasks[path] {
		rtask := task.Render(&listMetadata)
		listMetadata.tasks = append(listMetadata.tasks, rtask)
	}
	idColors := colorizeIds(listMetadata.idList)
	for _, task := range listMetadata.tasks {
		for _, tk := range task.tokens {
			if tk.token.Type == TokenID {
				tk.color = idColors[tk.token.Value.(int)]
			}
		}
	}
	formatPriorities(listMetadata.tasks)
	sessionMetadata.maxLen = max(sessionMetadata.maxLen, listMetadata.maxLen)
	sessionMetadata.idLen = max(sessionMetadata.idLen, listMetadata.idLen)
	sessionMetadata.countLen = max(sessionMetadata.countLen, listMetadata.countLen)
	sessionMetadata.doneCountLen = max(sessionMetadata.doneCountLen, sessionMetadata.doneCountLen)
	return nil
}

func PrintLists(paths []string, maxLen int) error {
	readTemporalFormatFallback()
	var err error
	for ndx := range paths {
		paths[ndx], err = prepFileTaskFromPath(paths[ndx])
		if err != nil {
			return err
		}
	}
	sessionMetadata := rPrint{lists: make(map[string]*rList)}
	for _, path := range paths {
		err := RenderList(&sessionMetadata, path)
		if err != nil {
			return err
		}
	}

	sessionMetadata.maxLen = min(sessionMetadata.maxLen, maxLen)
	for _, path := range paths {
		list := sessionMetadata.lists[path]
		list.maxLen = sessionMetadata.maxLen
		list.idLen = sessionMetadata.idLen
		list.countLen = sessionMetadata.countLen
		list.doneCountLen = sessionMetadata.doneCountLen
		for _, task := range list.tasks {
			task.idLen = list.idLen
			task.countLen = list.countLen
			task.doneCountLen = list.doneCountLen
		}
	}
	var out strings.Builder
	for _, path := range paths {
		list := sessionMetadata.lists[path]

		emptyCatThere := false
		categories := make(map[string]bool)
		for _, task := range list.tasks {
			if task.tsk.DoneCount > 0 {
				if task.tsk.Category == "" {
					emptyCatThere = true
				}
				categories[task.tsk.Category] = true
			}
		}
		useCatHeader := !((len(categories) == 1 && emptyCatThere) || len(categories) == 0)
		var lastCat string
		firstNonCat := true

		out.WriteString(formatListHeader(list))
		for _, task := range list.tasks {
			if useCatHeader && task.tsk.DoneCount > 0 && task.tsk.Category != lastCat {
				out.WriteString(formatCategoryHeader(task.tsk.Category, list))
				lastCat = task.tsk.Category
			}
			if useCatHeader && task.tsk.DoneCount == 0 && firstNonCat {
				firstNonCat = false
				out.WriteString(formatCategoryHeader("", list))
			}

			out.WriteString(task.stringify(true, true))
			out.WriteRune('\n')
		}
		out.WriteRune('\n')
	}
	fmt.Print(out.String())
	return nil
}

// single task
func PrintTask(id int, path string) error {
	readTemporalFormatFallback()
	path, err := prepFileTaskFromPath(path)
	if err != nil {
		return err
	}
	task, err := getTaskFromId(id, path)
	if err != nil {
		return err
	}
	fmt.Println(task.Raw())
	return nil
}
