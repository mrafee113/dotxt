package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"maps"
	"math"
	"path/filepath"
	"slices"
	"sort"
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
	colorStartHue := viper.GetFloat64("print.priority.start-hue")
	colorEndHue := viper.GetFloat64("print.priority.end-hue")

	taskToRTask := func() map[*Task]*rTask {
		out := make(map[*Task]*rTask)
		for _, t := range tasks {
			if t.tsk != nil {
				out[t.tsk] = t
			}
		}
		return out
	}()

	assignColor := func(task *rTask, color string) {
		for _, tk := range task.tokens {
			if tk.token != nil && tk.token.Type == TokenPriority {
				tk.color = color
				break
			}
		}
	}

	sortByPriority := func(tasks []*rTask) []*rTask {
		slices.SortFunc(tasks, func(l, r *rTask) int {
			if r := sortPriority(l.tsk, r.tsk); r != 2 {
				return r
			}
			return 0
		})
		return tasks
	}

	filterPriority := func(rts []*rTask) []*rTask {
		var out []*rTask
		for _, rt := range rts {
			if rt.tsk.Priority != nil && *rt.tsk.Priority != "" {
				out = append(out, rt)
			}
		}
		return out
	}

	var assignHue func(tasks []*rTask, startHue, endHue float64, depth int)
	assignHue = func(tasks []*rTask, startHue, endHue float64, depth int) {
		tasks = filterPriority(tasks)
		n := len(tasks)
		maxDepth := 0
		for _, t := range tasks {
			maxDepth = max(maxDepth, len(*t.tsk.Priority))
		}
		if n == 0 || depth > maxDepth || int(endHue-startHue) < n {
			return
		}

		groups := make(map[string][]*rTask)
		prefixes := []string{}
		for _, rt := range tasks {
			prio := *rt.tsk.Priority
			prefix := prio
			if len(prio) > depth {
				prefix = prio[:depth+1]
			}
			if _, exists := groups[prefix]; !exists {
				prefixes = append(prefixes, prefix)
			}
			groups[prefix] = append(groups[prefix], rt)
		}
		sort.Strings(prefixes)

		weights := make([]int, len(prefixes))
		for i, prefix := range prefixes {
			var countTasks func([]*Task) int
			countTasks = func(ts []*Task) int {
				count := 0
				for _, t := range ts {
					if t.Priority != nil && *t.Priority != "" {
						count++
					}
					count += countTasks(t.Children)
				}
				return count
			}
			for _, rt := range groups[prefix] {
				weights[i] += 1 + countTasks(rt.tsk.Children)
			}
		}

		totalWeight := 0
		for _, w := range weights {
			totalWeight += w
		}

		currentHue := startHue
		for i, prefix := range prefixes {
			group := groups[prefix]
			slices.SortFunc(group, func(a, b *rTask) int {
				return strings.Compare(*a.tsk.Priority, *b.tsk.Priority)
			})

			ratio := float64(weights[i]) / float64(totalWeight)
			nextHue := currentHue + (endHue-startHue)*ratio
			ng := len(group)

			for ndx, rt := range group {
				if len(*rt.tsk.Priority) <= depth {
					h := currentHue + (float64(ndx)+0.5)/float64(ng)*(nextHue-currentHue)
					assignColor(rt, utils.HslToHex(h, colorSaturation, colorLightness))
				}
				children := []*rTask{}
				for _, child := range rt.tsk.Children {
					if crt, ok := taskToRTask[child]; ok {
						children = append(children, crt)
					}
				}
				assignHue(children, currentHue, nextHue, 0)
			}

			assignHue(group, currentHue, nextHue, depth+1)
			currentHue = nextHue
		}
	}

	roots := []*rTask{}
	for _, rt := range tasks {
		if rt.tsk != nil && rt.tsk.Parent == nil {
			roots = append(roots, rt)
		}
	}

	assignHue(sortByPriority(roots), colorStartHue, colorEndHue, 0)
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
		startSaturation := viper.GetFloat64("print.progress.percentage.start-saturation")
		endSaturation := viper.GetFloat64("print.progress.percentage.end-saturation")
		lightness := viper.GetFloat64("print.progress.percentage.lightness")
		saturation := startSaturation + (endSaturation-startSaturation)*math.Pow(float64(percent)/100, 2.0)
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

func colorizeIds(ids map[string]bool) map[string]string {
	out := make(map[string]string)
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
	var idCollapse string
	if strings.HasPrefix(tk.raw, "$-id=") {
		idCollapse = "-"
	}
	return fmt.Sprintf("$%s%s=%s", idCollapse, tk.Key, *tk.Value.(*string))
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
		listMetadata = &rList{idList: make(map[string]bool)}
	}
	out := rTask{tsk: t, id: *t.ID, idColor: "print.color-index"}
	addAsRegular := func(token *Token) {
		out.tokens = append(out.tokens, &rToken{token: token, raw: token.String(t)})
	}
	specialTokenMap := func() map[string]*Token {
		out := make(map[string]*Token)
		t.Tokens.Filter(func(tk *Token) bool {
			switch tk.Type {
			case TokenDate, TokenID, TokenDuration, TokenPriority, TokenProgress:
				return true
			default:
				return false
			}
		}).ForEach(func(tk *Token) {
			out[tk.Key] = tk
		})
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
	t.Tokens.ForEach(func(tk *Token) {
		switch tk.Type {
		case TokenPriority, TokenProgress:
			return
		case TokenText:
			out.tokens = append(out.tokens, &rToken{
				token: tk, raw: tk.String(t),
			})
		case TokenID:
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   formatID(*tk),
				color: "",
			})
			listMetadata.idList[*tk.Value.(*string)] = true
		case TokenHint:
			var color string
			switch tk.Key {
			case "@":
				color = "print.color-at"
			case "#":
				color = "print.color-tag"
			case "+":
				color = "print.color-plus"
			}
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   tk.String(t), color: color,
			})
		case TokenDate:
			if slices.Contains([]string{"due", "end", "dead"}, tk.Key) {
				val, err := t.Time.getField(tk.Key)
				if err != nil {
					addAsRegular(tk)
					return
				}
				relStr, ok := temporalFormatFallback[tk.Key]
				if !ok {
					addAsRegular(tk)
					return
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(tk)
					return
				}
				color := "print.color-date-" + tk.Key
				if t.Time.DueDate != nil && t.Time.DueDate.Sub(rightNow) <= 0 {
					if tk.Key == "due" {
						color = "print.color-burnt"
					}
					if t.Time.Deadline != nil && t.Time.Deadline.Sub(rightNow) > 0 &&
						tk.Key == "dead" {
						color = "print.color-imminent-deadline"
					}
					if t.Time.EndDate != nil && t.Time.EndDate.Sub(rightNow) > 0 &&
						tk.Key == "end" {
						color = "print.color-running-event"
					}
				}
				out.tokens = append(out.tokens, &rToken{
					token: tk,
					raw:   fmt.Sprintf("$%s=%s", tk.Key, formatAbsoluteDatetime(val, rel)),
					color: color,
				})
			}
			if strings.HasPrefix(tk.Key, "r") {
				if reminderCount >= len(t.Time.Reminders) {
					addAsRegular(tk)
					return
				}
				val := t.Time.Reminders[reminderCount]
				reminderCount++
				relStr, ok := temporalFormatFallback["r"]
				if !ok {
					addAsRegular(tk)
					return
				}
				rel, err := t.Time.getField(relStr)
				if err != nil {
					addAsRegular(tk)
					return
				}
				if val.Sub(rightNow) < 0 {
					return
				}
				out.tokens = append(out.tokens, &rToken{
					token: tk,
					raw:   fmt.Sprintf("$r=%s", formatAbsoluteDatetime(val, rel)),
					color: "print.color-date-r",
				})
			}
		case TokenDuration:
			out.tokens = append(out.tokens, &rToken{
				token: tk,
				raw:   fmt.Sprintf("$every=%s", formatDuration(t.Time.Every)),
				color: "print.color-every",
			})
		default:
			addAsRegular(tk)
		}
	})
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

	Lists.Sort(path)
	listMetadata := rList{path: path, idList: make(map[string]bool)}
	sessionMetadata.lists[path] = &listMetadata
	for _, task := range Lists[path].Tasks {
		if task.ParentCollapsed() {
			continue
		}
		rtask := task.Render(&listMetadata)
		listMetadata.tasks = append(listMetadata.tasks, rtask)
	}
	idColors := colorizeIds(listMetadata.idList)
	for _, task := range listMetadata.tasks {
		for _, tk := range task.tokens {
			if tk.token.Type == TokenID {
				tk.color = idColors[*tk.token.Value.(*string)]
				if tk.dominantColor == "" &&
					((tk.token.Key == "id" && len(task.tsk.Children) == 0) ||
						(tk.token.Key == "P" && task.tsk.Parent == nil)) {
					tk.dominantColor = "print.color-dead-relations"
				}
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
