package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
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
				if hour < 2: 1h30M25s
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

	// this function assumes that all formats are .1f or .0f
	fmtFloat := func(fpoints int, format string, val float64) string {
		valD := decimal.NewFromFloat(val)
		if fpoints != -1 {
			valD = valD.Truncate(int32(fpoints))
		}
		if valD.IsZero() {
			return ""
		}
		return valD.String() + format
	}

	if totalSec == 0 {
		return "rn"
	}
	var dtStr string = func() string {
		if years := totalSec / secPerYr; 1.25 <= years {
			return fmtFloat(1, "y", years)
		}
		if years := totalSec / secPerYr; 1 <= years && years < 1.25 {
			return fmtFloat(0, "y", years) + fmtFloat(1, "m", math.Mod(totalSec, secPerYr)/secPerMo)
		}
		if months := totalSec / secPerMo; 2 <= months {
			return fmtFloat(1, "m", months)
		}
		if months := totalSec / secPerMo; 1 <= months && months < 2 {
			if weeks := math.Mod(totalSec, secPerMo) / secPerWeek; 1 <= weeks {
				return fmtFloat(0, "m", months) + fmtFloat(1, "w", weeks)
			}
			return fmtFloat(0, "m", months) + fmtFloat(0, "d", math.Mod(totalSec, secPerMo)/secPerDay)
		}
		if weeks := totalSec / secPerWeek; 1 <= weeks {
			return fmtFloat(0, "w", weeks) + fmtFloat(0, "d", math.Mod(totalSec, secPerWeek)/secPerDay)
		}
		if days := totalSec / secPerDay; 2 <= days {
			return fmtFloat(0, "d", days)
		}
		if days := totalSec / secPerDay; 1 <= days && days < 2 {
			return fmtFloat(0, "d", days) + fmtFloat(0, "'", math.Mod(totalSec, secPerDay)/secPerHr)
		}
		if hours := totalSec / secPerHr; 2 <= hours {
			return fmtFloat(0, "'", hours) + fmtFloat(0, `"`, math.Mod(totalSec, secPerHr)/secPerMin)
		}
		hours := totalSec / secPerHr
		mins := math.Mod(totalSec, secPerHr) / secPerMin
		secs := math.Mod(math.Mod(totalSec, secPerHr), secPerMin)
		return fmtFloat(0, "'", hours) + fmtFloat(0, `"`, mins) + fmtFloat(0, "s", secs)
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

func formatPriorities(tasks []*rTask) {
	if len(tasks) == 0 {
		return
	}

	colorSaturation := viper.GetFloat64("print.priority.saturation")
	colorLightness := viper.GetFloat64("print.priority.lightness")
	maxDepth := viper.GetInt("print.priority.group-depth")
	colorStartHue := viper.GetFloat64("print.priority.start-hue")
	colorEndHue := viper.GetFloat64("print.priority.end-hue")

	assignColor := func(task *rTask, color string) {
		for _, tk := range task.tokens {
			if tk.token != nil && tk.token.Type == TokenPriority {
				tk.color = color
				break
			}
		}
	}

	var assignHue func([]*rTask, float64, float64, int)
	assignHue = func(tasks []*rTask, startHue, endHue float64, depth int) {
		n := len(tasks)
		if n == 1 || depth >= maxDepth {
			for i, tsk := range tasks {
				h := startHue + (float64(i)+0.5)/float64(n)*(endHue-startHue)
				assignColor(tsk, utils.HslToHex(h, colorSaturation, colorLightness))
			}
			return
		}

		groups := make([][]*rTask, 0, n)
		prefixes := make([]string, 0, n)
		for _, t := range tasks {
			p := ""
			if len(*t.tsk.Priority) > depth {
				p = (*t.tsk.Priority)[:depth+1]
			}
			if len(groups) == 0 || prefixes[len(prefixes)-1] != p {
				groups = append(groups, []*rTask{t})
				prefixes = append(prefixes, p)
			} else {
				groups[len(groups)-1] = append(groups[len(groups)-1], t)
			}
		}

		if len(groups) == 1 {
			assignHue(tasks, startHue, endHue, depth+1)
			return
		}

		span := (endHue - startHue) / float64(len(groups))
		for i, grp := range groups {
			h0 := startHue + float64(i)*span
			h1 := h0 + span
			assignHue(grp, h0, h1, depth+1)
		}
	}

	assignHue(func() []*rTask {
		var out []*rTask
		for _, t := range tasks {
			if t.tsk != nil && t.tsk.Priority != nil && *t.tsk.Priority != "" {
				out = append(out, t)
			}
		}
		slices.SortFunc(out, func(l, r *rTask) int {
			if r := sortPriority(l.tsk, r.tsk); r != 2 {
				return r
			}
			return 0
		})
		return out
	}(), colorStartHue, colorEndHue, 0)
}

func formatProgress(p *Progress, countLen, doneCountLen int) []rToken {
	colorizePercentage := func(percent int) string {
		if percent < 0 {
			percent = 0
		}
		if percent > 100 {
			percent = 100
		}
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
	out.WriteString(" | ")
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
		if validateHexColor(color) != nil {
			color = viper.GetString("print.color-default")
		}
	}
	return color
}

func colorize(color, text string) string {
	return utils.Colorize(resolvColor(color), text)
}

func colorizeToken(raw, color, dominantColor string) string {
	if dominantColor != "" {
		return colorize(dominantColor, raw)
	}
	return colorize(color, raw)
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
	return fmt.Sprintf("$%s=%d", tk.Key, *tk.Value.(*int))
}

func formatCategoryHeader(category string, list *rList) string {
	var out strings.Builder
	out.WriteString(strings.Repeat(" ", list.idLen+1+
		list.countLen+1+list.doneCountLen+len("(100%) ")+
		viper.GetInt("print.progress.bartext-len")+1+
		-len(category)-1,
	))
	out.WriteString(fmt.Sprintf("%s ", category))
	out.WriteString(strings.Repeat("—", list.maxLen-out.Len()))
	out.WriteRune('\n')
	return colorize("print.progress.header", out.String())
}

func (t *Task) Render(listMetadata *rList) *rTask {
	if listMetadata == nil {
		listMetadata = &rList{idList: make(map[int]bool)}
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
				out[t.Tokens[ndx].Key] = t.Tokens[ndx]
			}
		}
		return out
	}()

	if t.Prog != nil {
		out.tokens = append(out.tokens, &rToken{token: specialTokenMap["p"]})
		listMetadata.countLen = max(listMetadata.countLen, len(strconv.Itoa(t.Prog.Count)))
		listMetadata.doneCountLen = max(listMetadata.doneCountLen, len(strconv.Itoa(t.Prog.DoneCount)))
	}
	if t.Priority != nil && *t.Priority != "" {
		out.tokens = append(out.tokens, &rToken{
			token: specialTokenMap["priority"],
			raw:   fmt.Sprintf("(%s)", *t.Priority),
		})
	}

	var dominantColor, defaultColor string = func() (string, string) {
		if t.Time.DueDate != nil && t.Time.DueDate.Sub(rightNow) <= 0 {
			if t.Time.EndDate == nil && t.Time.Deadline == nil {
				return "print.color-burnt", ""
			}
			if t.Time.EndDate != nil {
				if t.Time.EndDate.Sub(rightNow) <= 0 {
					return "print.color-burnt", ""
				}
				return "", "print.color-running-event-text"
			}
			if t.Time.Deadline != nil {
				if t.Time.Deadline.Sub(rightNow) <= 0 {
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
				token: t.Tokens[ndx], raw: t.Tokens[ndx].Raw,
			})
		case TokenID:
			out.tokens = append(out.tokens, &rToken{
				token: t.Tokens[ndx],
				raw:   formatID(*t.Tokens[ndx]),
				color: "",
			})
			listMetadata.idList[*t.Tokens[ndx].Value.(*int)] = true
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
				token: t.Tokens[ndx],
				raw:   t.Tokens[ndx].Raw, color: color,
			})
		case TokenDate:
			if slices.Contains([]string{"due", "end", "dead"}, t.Tokens[ndx].Key) {
				val, err := t.Time.getField(t.Tokens[ndx].Key)
				if err != nil {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				relStr, ok := temporalFormatFallback[t.Tokens[ndx].Key]
				if !ok {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				color := "print.color-date-" + t.Tokens[ndx].Key
				if t.Time.DueDate != nil && t.Time.DueDate.Sub(rightNow) <= 0 {
					if t.Tokens[ndx].Key == "due" {
						color = "print.color-burnt"
					}
					if t.Time.Deadline != nil && t.Time.Deadline.Sub(rightNow) > 0 &&
						t.Tokens[ndx].Key == "dead" {
						color = "print.color-imminent-deadline"
					}
					if t.Time.EndDate != nil && t.Time.EndDate.Sub(rightNow) > 0 &&
						t.Tokens[ndx].Key == "end" {
						color = "print.color-running-event"
					}
				}
				out.tokens = append(out.tokens, &rToken{
					token: t.Tokens[ndx],
					raw:   fmt.Sprintf("$%s=%s", t.Tokens[ndx].Key, formatAbsoluteDatetime(val, rel)),
					color: color,
				})
			}
			if strings.HasPrefix(t.Tokens[ndx].Key, "r") {
				if reminderCount >= len(t.Time.Reminders) {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				val := t.Time.Reminders[reminderCount]
				reminderCount++
				relStr, ok := temporalFormatFallback["r"]
				if !ok {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(t.Tokens[ndx])
					continue
				}
				if val.Sub(rightNow) < 0 {
					continue
				}
				out.tokens = append(out.tokens, &rToken{
					token: t.Tokens[ndx],
					raw:   fmt.Sprintf("$r=%s", formatAbsoluteDatetime(val, rel)),
					color: "print.color-date-r",
				})
			}
		case TokenDuration:
			out.tokens = append(out.tokens, &rToken{
				token: t.Tokens[ndx],
				raw:   fmt.Sprintf("$every=%s", formatDuration(t.Time.Every)),
				color: "print.color-every",
			})
		default:
			addAsRegular(t.Tokens[ndx])
		}
	}
	listMetadata.maxLen = max(listMetadata.maxLen, len(out.stringify(false, -1)))
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
				tk.color = idColors[*tk.token.Value.(*int)]
			}
		}
	}
	formatPriorities(listMetadata.tasks)
	sessionMetadata.maxLen = max(sessionMetadata.maxLen, listMetadata.maxLen)
	sessionMetadata.idLen = max(sessionMetadata.idLen, listMetadata.idLen)
	sessionMetadata.countLen = max(sessionMetadata.countLen, listMetadata.countLen)
	sessionMetadata.doneCountLen = max(sessionMetadata.doneCountLen, listMetadata.doneCountLen)
	return nil
}

func PrintLists(paths []string, maxLen, minlen int) error {
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
	sessionMetadata.maxLen = max(sessionMetadata.maxLen, minlen)
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
			if task.tsk.Prog != nil {
				if task.tsk.Prog.Category == "" {
					emptyCatThere = true
				}
				categories[task.tsk.Prog.Category] = true
			}
		}
		useCatHeader := !((len(categories) == 1 && emptyCatThere) || len(categories) == 0)
		var lastCat string
		firstNonCat := true

		out.WriteString(formatListHeader(list))
		for _, task := range list.tasks {
			if useCatHeader && task.tsk.Prog != nil && task.tsk.Prog.Category != lastCat {
				cat := task.tsk.Prog.Category
				if cat == "" {
					cat = "*"
				}
				out.WriteString(formatCategoryHeader(cat, list))
				lastCat = task.tsk.Prog.Category
			}
			if useCatHeader && task.tsk.Prog == nil && firstNonCat {
				firstNonCat = false
				out.WriteString(formatCategoryHeader("", list))
			}

			out.WriteString(task.stringify(true, sessionMetadata.maxLen))
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
