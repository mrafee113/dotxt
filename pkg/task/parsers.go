package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"math"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/viper"
)

/*
ommitted fields will be replaced from RightNow, I think
%Y:2006, %y:06, %m:01, %d:02, %H:15, %M:04, %S:05, %b:Jan

datetime: [date][[time]]
time:

	3-parter: T%H-%M-%S
	2-parter:
		T<a>-<b>:
			if a <= 23: T%H-%M
			else:		T%M-%S
	1-parter:
		T<a>:
			if a <= 23: T%H
			else: 		T%M

date:

	3-parter:
		%Y-%m-%d
		%y-%m-%d
		%Y-%b-%d
		%y-%b-%d
	2-parter:
		<a>-<b>:
			if a >= 13 || <b> == %b:
				- %Y-%m
				- %Y-%b
				- %y-%m
				- %y-%b
			else:
				- %m-%d
				- %b-%d
	1-parter:
		<a>:
			if a >= 13:
				- %Y
				- %y
			else:
				- %m
				- %b
*/
func parseAbsoluteDatetime(absDt string) (*time.Time, error) {
	if absDt == "" {
		return nil, fmt.Errorf("%w: empty", terrors.ErrParse)
	}
	var timeStr string
	if ndx := strings.IndexRune(absDt, 'T'); ndx != -1 {
		timeStr = absDt[ndx+1:]
		absDt = absDt[:ndx]
		if len(timeStr) == 0 {
			return nil, fmt.Errorf("%w: invalid use of T", terrors.ErrParse)
		}
		parts := strings.Split(timeStr, "-")
		n := len(parts)
		if n == 0 || n > 3 {
			return nil, fmt.Errorf("%w: invalid use T: either no string afterwards or too many dashes", terrors.ErrParse)
		}
		vals := make([]int, n)
		for ndx := range parts {
			val, err := strconv.Atoi(parts[ndx])
			if err != nil {
				return nil, fmt.Errorf("%w: %w: invalid hour, minute or second: %w", terrors.ErrParse, terrors.ErrValue, err)
			}
			if val >= 60 {
				return nil, fmt.Errorf("%w: %w: invalid hour, minute or second value: %d", terrors.ErrParse, terrors.ErrValue, val)
			}
			vals[ndx] = val
			parts[ndx] = fmt.Sprintf("%02s", parts[ndx])
		}
		switch n {
		case 3:
			if vals[0] >= 24 {
				return nil, fmt.Errorf("%w: %w: invalid hour value: %d", terrors.ErrParse, terrors.ErrValue, vals[0])
			}
			timeStr = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		case 2:
			if vals[0] <= 23 {
				timeStr = fmt.Sprintf("%s-%s-00", parts[0], parts[1])
			} else {
				timeStr = fmt.Sprintf("00-%s-%s", parts[0], parts[1])
			}
		case 1:
			if vals[0] <= 23 {
				timeStr = fmt.Sprintf("%s-00-00", parts[0])
			} else {
				timeStr = fmt.Sprintf("00-%s-00", parts[0])
			}
		}
	} else {
		timeStr = "00-00-00"
	}

	var dateStr string
	if len(absDt) > 0 {
		parts := strings.Split(absDt, "-")
		n := len(parts)
		if n == 0 || n > 3 {
			return nil, fmt.Errorf("%w: invalid use of date: too many dashes", terrors.ErrParse)
		}
		vals := make([]int, n)
		for ndx := range parts {
			val, err := strconv.Atoi(parts[ndx])
			if err == nil {
				if val >= 3000 {
					return nil, fmt.Errorf("%w: %w: invalid year, month or day value: %d",
						terrors.ErrParse, terrors.ErrValue, val)
				}
				vals[ndx] = val
			} else {
				t, err := time.Parse("Jan", parts[ndx])
				if err != nil {
					return nil, fmt.Errorf("%w: %w: invalid date value which is neither 'Jan' nor a number: %s",
						terrors.ErrParse, terrors.ErrValue, parts[ndx])
				}
				vals[ndx] = math.MaxInt
				parts[ndx] = t.Format("01")
			}
			parts[ndx] = fmt.Sprintf("%02s", parts[ndx])
		}
		handleSmallYear := func(year string) string {
			t, _ := time.Parse("06", year)
			return t.Format("2006")
		}
		switch n {
		case 3:
			if vals[0] < 100 {
				parts[0] = handleSmallYear(parts[0])
			}
			dateStr = fmt.Sprintf("%s-%s-%s", parts[0], parts[1], parts[2])
		case 2:
			if (vals[0] >= 13 && vals[0] < 3000) || vals[1] == math.MaxInt {
				if vals[0] < 100 {
					parts[0] = handleSmallYear(parts[0])
				}
				dateStr = fmt.Sprintf("%s-%s-01", parts[0], parts[1])
			} else {
				dateStr = fmt.Sprintf("%s-%s-%s", rightNow.Format("2006"), parts[0], parts[1])
			}
		case 1:
			if vals[0] >= 13 && vals[0] < 3000 {
				if vals[0] < 100 {
					parts[0] = handleSmallYear(parts[0])
				}
				dateStr = fmt.Sprintf("%s-01-01", parts[0])
			} else {
				dateStr = fmt.Sprintf("%s-%s-01", rightNow.Format("2006"), parts[0])
			}
		}
	} else {
		dateStr = fmt.Sprintf("%s-01-01", rightNow.Format("2006"))
	}

	t, err := time.ParseInLocation("2006-01-02T15-04-05", fmt.Sprintf("%sT%s", dateStr, timeStr), time.Local)
	if err != nil {
		return nil, err
	}
	return &t, nil
}

/*
datetime: [date]T[[time]]
date:

	if month == 1 and day == 1: %Y
	else if day == 1:			%Y-%m
	else: 						%Y-%m-%d

time:

	if hour == 0 and minute == 0 and second == 0:
	else if minute == 0 and second == 0:       	  T%H
	else if second == 0:						  T%H-%M
	else:										  T%H-%M-%S
*/
func unparseAbsoluteDatetime(absDt time.Time) string {
	var dateStr string
	if absDt.Month() == 1 && absDt.Day() == 1 {
		dateStr = absDt.Format("2006")
	} else if absDt.Day() == 1 {
		dateStr = absDt.Format("2006-01")
	} else {
		dateStr = absDt.Format("2006-01-02")
	}
	var timeStr string
	if absDt.Hour() == 0 && absDt.Minute() == 0 && absDt.Second() == 0 {
	} else if absDt.Minute() == 0 && absDt.Second() == 0 {
		timeStr = absDt.Format("15")
	} else if absDt.Second() == 0 {
		timeStr = absDt.Format("15-04")
	} else {
		timeStr = absDt.Format("15-04-05")
	}
	if len(timeStr) > 0 {
		timeStr = "T" + timeStr
	}
	return fmt.Sprintf("%s%s", dateStr, timeStr)
}

func parseDuration(dur string) (*time.Duration, error) {
	if err := validateEmptyText(dur); err != nil {
		return nil, err
	}
	sign := 1
	if dur[0] == '-' {
		sign = -1
		dur = dur[1:]
	} else if dur[0] == '+' {
		dur = dur[1:]
	}
	if dur == "0" {
		return utils.MkPtr(time.Duration(0)), nil
	}

	const day = 24 * time.Hour
	var duration float64
	var numStr string
	for _, char := range dur {
		if unicode.IsDigit(char) || char == '.' {
			numStr += string(char)
			continue
		}
		num, err := strconv.ParseFloat(numStr, 64)
		if err != nil {
			return nil, fmt.Errorf("%w: number conversion of '%s' failed: %w", terrors.ErrParse, numStr, err)
		}
		var multiplier time.Duration
		switch char {
		case 'y':
			multiplier = 365 * day
		case 'm':
			multiplier = 30 * day
		case 'w':
			multiplier = 7 * day
		case 'd':
			multiplier = day
		case 'h':
			multiplier = time.Hour
		case 'M':
			multiplier = time.Minute
		case 's':
			multiplier = time.Second
		default:
			return nil, fmt.Errorf("%w: unexpected time unit %q", terrors.ErrParse, char)
		}
		duration += float64(multiplier) * num
		numStr = ""
	}
	if numStr != "" {
		return nil, fmt.Errorf("%w: trailing numbers without a time unit %q", terrors.ErrParse, numStr)
	}
	return utils.MkPtr(time.Duration(sign) * time.Duration(duration)), nil
}

func unparseDuration(dur time.Duration) string {
	totalSec := int(dur.Seconds())
	if totalSec == 0 {
		return "0s"
	}
	var sign string
	if totalSec < 0 {
		sign = "-"
		totalSec *= -1
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
		parts = append(parts, fmt.Sprintf("%dm", months))
	}
	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}
	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}
	if mins > 0 {
		parts = append(parts, fmt.Sprintf("%dM", mins))
	}
	if secs > 0 || len(parts) == 0 {
		parts = append(parts, fmt.Sprintf("%ds", secs))
	}

	return strings.Join(parts, "")
}

func unparseRelativeDatetime(dt, rel time.Time) string {
	return unparseDuration(dt.Sub(rel))
}

func (tk *Token) unparseRelativeDatetime(t *Temporal, val *time.Time) (string, error) {
	curDtTxt := strings.TrimPrefix(tk.Raw, fmt.Sprintf("$%s=", tk.Key))
	fallback, _, err := getTemporalFallback(tk.Key, curDtTxt)
	if err != nil {
		return "", err
	}
	if val == nil {
		val = tk.Value.(*time.Time)
	}
	rel, err := t.getField(fallback)
	if err != nil {
		return "", err
	}
	newDtTxt := unparseRelativeDatetime(*val, *rel)
	if temporalFallback[tk.Key] != fallback {
		newDtTxt = fmt.Sprintf("%s:%s", fallback, newDtTxt)
	}
	newDtTxt = fmt.Sprintf("$%s=%s", tk.Key, newDtTxt)
	return newDtTxt, nil
}

func getTemporalFallback(field, dt string) (string, string, error) {
	fallback, ok := temporalFallback[field]
	if !ok {
		return "", "", fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, field)
	}
	if ndx := strings.IndexRune(dt, ':'); ndx != -1 {
		if ndx == 0 || ndx == len(dt)-1 {
			return "", "", fmt.Errorf("%w: invalid use of var:dt", terrors.ErrParse)
		}
		fallback = dt[:ndx]
		_, ok := temporalFallback[fallback]
		if !ok {
			return "", "", fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, fallback)
		}
		dt = dt[ndx+1:]
	}
	return fallback, dt, nil
}

func parseTmpRelativeDatetime(field, dt string) (*temporalNode, error) {
	fallback, dt, err := getTemporalFallback(field, dt)
	if err != nil {
		return nil, err
	}
	duration, err := parseDuration(dt)
	if err != nil {
		return nil, fmt.Errorf("%w: failed parsing tmp relative datetime: %w", terrors.ErrParse, err)
	}
	return &temporalNode{Field: field, Ref: fallback, Offset: duration}, nil
}

func parseProgress(token string) (*Progress, error) {
	parts := strings.Split(token, "/")
	if len(parts) < 3 {
		return nil, fmt.Errorf("%w: $progress: number of '/' is less than 2: required fields not provided: %s", terrors.ErrParse, token)
	}
	unit, count, doneCount := parts[0], parts[1], parts[2]
	var category string
	switch len(parts) {
	case 4:
		category = parts[3]
	}
	if unit == "" {
		return nil, fmt.Errorf("%w: %w: $progress: unit is empty", terrors.ErrParse, terrors.ErrValue)
	}
	doneCountInt, err := strconv.Atoi(doneCount)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: $progress: doneCount to int: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	if doneCountInt < 1 {
		return nil, fmt.Errorf("%w: %w: $progress: doneCount minimum is 1: %d", terrors.ErrParse, terrors.ErrValue, doneCountInt)
	}
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: $progress: count to int: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	if countInt < 0 {
		return nil, fmt.Errorf("%w: %w: $progress: count minimum is 0: %d", terrors.ErrParse, terrors.ErrValue, countInt)
	}
	countInt = min(countInt, doneCountInt)
	return &Progress{
		Unit: unit, Category: category,
		Count: countInt, DoneCount: doneCountInt,
	}, nil
}

func unparseProgress(progress Progress) (string, error) {
	if progress.Unit == "" {
		return "", fmt.Errorf("%w: progress unit cannot be empty", terrors.ErrValue)
	}
	if progress.DoneCount <= 0 {
		return "", fmt.Errorf("%w: progress doneCount cannot be less than 1: %d", terrors.ErrValue, progress.DoneCount)
	}
	if progress.Count < 0 {
		return "", fmt.Errorf("%w: progress count cannot be less than 0: %d", terrors.ErrValue, progress.Count)
	}
	if progress.Count > progress.DoneCount {
		return "", fmt.Errorf("%w: progress count cannot be greater than doneCount: %d > %d", terrors.ErrValue, progress.Count, progress.DoneCount)
	}
	base := fmt.Sprintf("%s/%d/%d", progress.Unit, progress.Count, progress.DoneCount)
	if progress.Category != "" {
		base += "/" + progress.Category
	}
	return base, nil
}

func parsePriority(line string) (int, int, error) {
	if len(line) == 0 {
		return -1, -1, terrors.ErrEmptyText
	}
	if line[0] != '(' {
		return -1, -1, fmt.Errorf("%w: %w: (", terrors.ErrParse, terrors.ErrNotFound)
	}
	ndx := 1
	n := len(line)
	for ; ndx < n && line[ndx] != ' '; ndx++ {

	}
	ndx--
	if line[ndx] == ')' {
		return 1, ndx, nil
	} else {
		return -1, -1, fmt.Errorf("%w: %w: priority", terrors.ErrParse, terrors.ErrNotFound)
	}
}

func resolveDates(tokens []*Token) []error {
	tokenKeyToNdx := make(map[string]int)
	nodes := make(map[string]*temporalNode)
	resolved := make(map[string]time.Time)
	resolved["rn"] = rightNow
	var totalDateCount int
	var reminderKeys []string
	for ndx, token := range tokens {
		if token.Type != TokenDate {
			continue
		}
		if token.Key == "r" {
			token.Key = fmt.Sprintf("r%d", ndx)
		}
		totalDateCount++
		tokenKeyToNdx[token.Key] = ndx
		switch v := token.Value.(type) {
		case *time.Time:
			resolved[token.Key] = *v
		case *temporalNode:
			if v.Field == "r" {
				v.Field = fmt.Sprintf("r%d", ndx)
				reminderKeys = append(reminderKeys, v.Field)
			}
			nodes[token.Key] = v
		}
	}
	for len(resolved)-1 < totalDateCount {
		changed := false
		// this order is based on temporalFallback and please review this if you change that
		for _, key := range append([]string{"c", "lud", "due", "end", "dead"}, reminderKeys...) {
			n, ok := nodes[key]
			if !ok {
				continue
			}
			if _, ok := resolved[n.Field]; ok {
				continue
			}
			ref := n.Ref
			for range 3 {
				if base, ok := resolved[ref]; ok {
					resolved[n.Field] = base.Add(*n.Offset)
					changed = true
				} else {
					tmp, ok := temporalFallback[ref]
					if !ok {
						continue
					}
					ref = tmp
				}
			}
		}
		if !changed {
			return []error{fmt.Errorf("%w: dependency of date fields are not resolvable", terrors.ErrValue)}
		}
	}
	var errs []error
	for key, ndx := range tokenKeyToNdx {
		tVal, ok := resolved[key]
		if ok {
			tokens[ndx].Value = &tVal
		} else {
			errs = append(errs, fmt.Errorf("%w: somehow the date '%s' was not resolved", terrors.ErrNotFound, key))
		}
	}
	for _, tk := range tokens {
		if tk.Type == TokenDate && strings.HasPrefix(tk.Key, "r") {
			tk.Key = "r"
		}
	}
	return errs
}

func dateToTextToken(dt *Token) {
	dt.Key = ""
	dt.Type = TokenText
	dt.Value = dt.Raw
}

/*
tokenizeLine splits a line into tokens with the following rules:
  - A single unescaped space is just a separator.  Extra spaces beyond the first
    become explicit " " tokens (e.g. "a  b" → ["a","b"," "]).
  - "\ " produces a literal space in the token.
  - Double, single, or back-tick quotes group everything (including spaces) until
    the matching closing quote; inside them only \" (or \' or \`) is special.
    The opening and closing characters of \" and \' are also kept intact in the token;
    but the \` are stripped away.
  - If a quote never closes, we abandon the quote and treat the rest as plain text.
  - If a \; is provided the token is forced to be split
  - The function replaces any \n with '\ '
*/
func tokenizeLine(line string) []string {
	line = strings.NewReplacer("\t", "    ", "\r", " ", "\n", " ", "\v", " ", "\f", " ").Replace(line)
	var tokens []string
	var cur strings.Builder
	rs := []rune(line)
	n := len(rs)

	flush := func() {
		if cur.Len() > 0 {
			tokens = append(tokens, cur.String())
			cur.Reset()
		}
	}

	for i := 0; i < n; {
		r := rs[i]

		if r == '\\' && i+1 < n && rs[i+1] == ' ' {
			cur.WriteRune(' ')
			i += 2
			continue
		}

		if r == '"' || r == '\'' || r == '`' {
			snapTokens := slices.Clone(tokens)
			snapStr := cur.String()
			snapI := i

			open := r
			if open != '`' {
				cur.WriteRune(open)
			}
			i++ // consume the opening quote
			for i < n {
				if rs[i] == '\\' && i+1 < n && rs[i+1] == open {
					cur.WriteRune(open)
					i += 2
					continue
				}
				if rs[i] == open {
					if open != '`' {
						cur.WriteRune(open)
					}
					i++ // consume closing
					break
				}
				cur.WriteRune(rs[i])
				i++
			}
			// check unterminated
			if i > n || (i <= n && rs[i-1] != open) {
				tokens = snapTokens
				cur.Reset()
				cur.WriteString(snapStr)
				i = snapI
			} else {
				flush()
				continue
			}
		}

		if r == ' ' {
			j := i
			for j < n && rs[j] == ' ' {
				j++
			}
			count := j - i - 1 // excluding the first one as separator
			flush()
			if count > 0 {
				tokens = append(tokens, strings.Repeat(" ", count))
			}
			i = j
			continue
		}

		if r == '\\' && i+1 < n && rs[i+1] == ';' {
			flush()
			i += 2
			continue
		}

		cur.WriteRune(r)
		i++
	}

	flush()
	return tokens
}

func parseTokens(line string) ([]*Token, []error) {
	specialFields := make(map[string]bool)
	var tokens []*Token
	var errs []error
	handleTokenText := func(tokenStr string, err error) {
		if err != nil {
			errs = append(errs, err)
		}
		tokens = append(tokens, &Token{Type: TokenText, Raw: tokenStr, Value: tokenStr})
	}
	if i, j, err := parsePriority(line); err == nil {
		p := line[i:j]
		tokens = append(tokens, &Token{
			Type: TokenPriority, Key: "priority",
			Value: &p,
			Raw:   fmt.Sprintf("(%s)", p),
		})
		line = line[j+1:]
	}
	for _, tokenStr := range tokenizeLine(line) {
		switch tokenStr[0] {
		case '+', '@', '#':
			if err := validateHint(tokenStr); err != nil {
				handleTokenText(tokenStr, nil)
				continue
			}
			tokens = append(tokens, &Token{
				Type: TokenHint, Raw: tokenStr,
				Key: tokenStr[0:1], Value: utils.MkPtr(tokenStr[1:]),
			})
		case '$':
			keyValue := strings.SplitN(tokenStr[1:], "=", 2)
			if len(keyValue) != 2 {
				errs = append(errs, fmt.Errorf("%w: zero or multiple `=` were found: %s", terrors.ErrParse, tokenStr))
				handleTokenText(tokenStr, nil)
				continue
			}
			key, value := keyValue[0], keyValue[1]
			if validateEmptyText(value) != nil {
				handleTokenText(tokenStr, nil)
				continue
			}
			_, seenKey := specialFields[key]
			if key != "r" && seenKey {
				continue
			} else if key != "r" {
				specialFields[key] = true
			}
			switch key {
			case "-id", "id", "P":
				k := strings.Replace(key, "-", "", 1)
				tokens = append(tokens, &Token{
					Type: TokenID, Raw: tokenStr,
					Key: k, Value: utils.MkPtr(value),
				})
			case "c", "lud", "due", "end", "dead", "r":
				var dt any
				dt, err := parseAbsoluteDatetime(value)
				if err != nil {
					dt, err = parseTmpRelativeDatetime(key, value)
					if err != nil {
						handleTokenText(tokenStr, fmt.Errorf("%w: $%s", err, key))
						continue
					}
				}
				tokens = append(tokens, &Token{
					Type: TokenDate, Raw: tokenStr,
					Key: key, Value: dt,
				})
			case "every":
				duration, err := parseDuration(value)
				oneDay := 24 * 60 * 60 * time.Second
				tenYears := 10 * 365 * 24 * 60 * 60 * time.Second
				if err != nil || *duration < oneDay || tenYears <= *duration {
					handleTokenText(tokenStr, fmt.Errorf("%w: $every: %w", terrors.ErrParse, err))
					continue
				}
				tokens = append(tokens, &Token{
					Type: TokenDuration, Raw: tokenStr,
					Key: key, Value: duration,
				})
			case "p":
				progress, err := parseProgress(value)
				if err != nil {
					handleTokenText(tokenStr, err)
					continue
				}
				tokens = append(tokens, &Token{
					Type: TokenProgress, Raw: tokenStr,
					Key: "p", Value: progress,
				})
			default:
				handleTokenText(tokenStr, nil)
			}
		default:
			handleTokenText(tokenStr, nil)
		}
	}
	tmpErrs := resolveDates(tokens)
	if len(tmpErrs) > 0 {
		errs = append(errs, tmpErrs...)
	}
	return tokens, errs
}

func ParseTask(id *int, line string) (*Task, error) {
	if err := validateEmptyText(line); err != nil {
		return nil, err
	}

	task := &Task{ID: id, Time: utils.MkPtr(Temporal{})}
	tokens, errs := parseTokens(line)
	for ndx := range tokens {
		token := tokens[ndx]
		switch token.Type {
		case TokenID:
			val := token.Value.(*string)
			switch token.Key {
			case "id":
				task.EID = val
				if strings.HasPrefix(token.Raw, "$-id") {
					task.EIDCollapse = true
				}
			case "P":
				task.PID = val
			}
		case TokenHint:
			task.Hints = append(task.Hints, utils.MkPtr(fmt.Sprintf("%s%s", token.Key, *token.Value.(*string))))
		case TokenPriority:
			task.Priority = token.Value.(*string)
		case TokenDate:
			switch token.Key {
			case "c":
				val := token.Value.(*time.Time)
				if val.After(rightNow) {
					dateToTextToken(tokens[ndx])
					continue
				}
				task.Time.CreationDate = val
			case "lud":
				val := token.Value.(*time.Time)
				if val.After(rightNow) {
					dateToTextToken(tokens[ndx])
					continue
				}
				task.Time.LastUpdated = val
			case "due":
				task.Time.DueDate = token.Value.(*time.Time)
			case "r":
				task.Time.Reminders = append(task.Time.Reminders, token.Value.(*time.Time))
			case "end":
				task.Time.EndDate = token.Value.(*time.Time)
			case "dead":
				task.Time.Deadline = token.Value.(*time.Time)
			}
		case TokenDuration:
			task.Time.Every = token.Value.(*time.Duration)
		case TokenProgress:
			task.Prog = token.Value.(*Progress)
		}
	}
	task.Tokens = tokens
	if task.Time.CreationDate == nil {
		task.Time.CreationDate = &rightNow
		task.Tokens = append(task.Tokens, &Token{
			Type: TokenDate, Raw: fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(rightNow)),
			Key: "c", Value: &rightNow,
		})
	}
	if task.Time.LastUpdated == nil {
		task.Time.LastUpdated = &rightNow
		ludVal := rightNow.Add(time.Second)
		task.Tokens = append(task.Tokens, &Token{
			Type: TokenDate, Raw: "$lud=" + unparseDuration(time.Duration(0)),
			Key: "lud", Value: &ludVal,
		})
	}
	findToken := func(tipe TokenType, key string) (*Token, int) {
		for ndx := range task.Tokens {
			if task.Tokens[ndx].Type == tipe && task.Tokens[ndx].Key == key {
				return task.Tokens[ndx], ndx
			}
		}
		return nil, -1
	}
	if task.Time.DueDate != nil && !task.Time.DueDate.After(*task.Time.CreationDate) {
		tk, _ := findToken(TokenDate, "due")
		if tk != nil {
			dateToTextToken(tk)
			task.Time.DueDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: due date token", terrors.ErrNotFound))
		}
	}
	if task.Time.Deadline != nil && (task.Time.DueDate == nil || !task.Time.Deadline.After(*task.Time.DueDate)) {
		tk, _ := findToken(TokenDate, "dead")
		if tk != nil {
			dateToTextToken(tk)
			task.Time.Deadline = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
	}
	if task.Time.EndDate != nil && (task.Time.DueDate == nil || !task.Time.EndDate.After(*task.Time.DueDate)) {
		tk, _ := findToken(TokenDate, "end")
		if tk != nil {
			dateToTextToken(tk)
			task.Time.EndDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	if task.Time.EndDate != nil && task.Time.Deadline != nil {
		tk, _ := findToken(TokenDate, "dead")
		if tk != nil {
			dateToTextToken(tk)
			task.Time.Deadline = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
		tk, _ = findToken(TokenDate, "end")
		if tk != nil {
			dateToTextToken(tk)
			task.Time.EndDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	if !task.Time.LastUpdated.After(*task.Time.CreationDate) {
		tk, _ := findToken(TokenDate, "lud")
		if tk != nil {
			tk.Raw = "$lud=" + unparseDuration(time.Duration(0))
			ludVal := task.Time.CreationDate.Add(time.Second)
			tk.Value = &ludVal
			task.Time.LastUpdated = &ludVal
		} else {
			errs = append(errs, fmt.Errorf("%w: lud date token", terrors.ErrNotFound))
		}
	}
	for ndx := len(task.Time.Reminders) - 1; ndx >= 0; ndx-- {
		if !task.Time.Reminders[ndx].After(*task.Time.CreationDate) {
			for ndxTk := range task.Tokens {
				if task.Tokens[ndxTk].Type == TokenDate &&
					strings.HasPrefix(task.Tokens[ndxTk].Key, "r") &&
					*task.Tokens[ndxTk].Value.(*time.Time) == *task.Time.Reminders[ndx] {
					task.Time.Reminders = slices.Delete(task.Time.Reminders, ndx, ndx+1)
					dateToTextToken(task.Tokens[ndxTk])
					break
				}
			}
		}
	}
	if viper.GetBool("debug") {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	return task, nil
}

func ParseTasks(filepath string) ([]*Task, error) {
	if !utils.FileExists(filepath) {
		return []*Task{}, os.ErrNotExist
	}
	data, err := os.ReadFile(filepath)
	if err != nil {
		return []*Task{}, err
	}
	lines := strings.Split(string(data), "\n")
	var tasks []*Task
	var errs error = fmt.Errorf("")
	for id, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		task, err := ParseTask(&id, line)
		if err != nil {
			errs = fmt.Errorf("%w\nline %d: %w", errs, id, err)
		}
		tasks = append(tasks, task)
	}
	return tasks, nil
}
