package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/spf13/viper"
)

func parseAbsoluteDatetime(absDt string) (*time.Time, error) {
	if strings.Count(absDt, "T") != 1 {
		return nil, fmt.Errorf("%w: datetime doesn't have 'T'", terrors.ErrParse)
	}
	var t time.Time
	var err error
	if dashCount := strings.Count(absDt, "-"); dashCount == 4 {
		t, err = time.ParseInLocation("2006-01-02T15-04-05", absDt, time.Local)
	} else if dashCount == 3 {
		t, err = time.ParseInLocation("2006-01-02T15-04", absDt, time.Local)
	} else {
		return nil, fmt.Errorf("%w: datetime doesn't satisfy 3 <= dashCount <= 4", terrors.ErrParse)
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func unparseAbsoluteDatetime(absDt time.Time) string {
	return absDt.Format("2006-01-02T15-04-05")
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
	var duration time.Duration
	var numStr string
	for _, char := range dur {
		if unicode.IsDigit(char) {
			numStr += string(char)
			continue
		}
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("%w: number conversion of '%s' failed", terrors.ErrParse, numStr)
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
		case 'S':
			multiplier = time.Second
		default:
			return nil, fmt.Errorf("%w: unexpected time unit %q", terrors.ErrParse, char)
		}
		duration += multiplier * time.Duration(num)
		numStr = ""
	}
	if numStr != "" {
		return nil, fmt.Errorf("%w: trailing numbers without a time unit %q", terrors.ErrParse, numStr)
	}
	duration *= time.Duration(sign)
	return &duration, nil
}

func unparseDuration(dur time.Duration) string {
	totalSec := int(dur.Seconds())
	if totalSec == 0 {
		return "0S"
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
		parts = append(parts, fmt.Sprintf("%dS", secs))
	}

	return strings.Join(parts, "")
}

func unparseRelativeDatetime(dt, rel time.Time) string {
	return unparseDuration(dt.Sub(rel))
}

func getTemporalFallback(field, dt string) (string, string, error) {
	fallback, ok := temporalFallback[field]
	if !ok {
		return "", "", fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, field)
	}
	if strings.HasPrefix(dt, "variable=") {
		ndx := strings.Index(dt, ";")
		if ndx == -1 {
			return "", "", fmt.Errorf("%w: did not find ';'", terrors.ErrParse)
		}
		fallback = dt[len("variable="):ndx]
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
	subTokens := strings.Split(token, "/")
	var unit, category string
	var count, doneCount string = "0", "0"
	if ln := len(subTokens); ln == 4 {
		unit, category = subTokens[0], subTokens[1]
		count, doneCount = subTokens[2], subTokens[3]
	} else if ln == 3 {
		unit = subTokens[0]
		count, doneCount = subTokens[1], subTokens[2]
	} else if ln == 2 {
		unit = subTokens[0]
		doneCount = subTokens[1]
	} else {
		return nil, fmt.Errorf("%w: $progress: number of `/` is either less than 2 or greater than 4: %s", terrors.ErrParse, token)
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
	if progress.Category != "" && progress.Count > 0 {
		return fmt.Sprintf(
			"%s/%s/%d/%d",
			progress.Unit, progress.Category,
			progress.Count, progress.DoneCount,
		), nil
	}
	if progress.Count > 0 {
		return fmt.Sprintf(
			"%s/%d/%d",
			progress.Unit,
			progress.Count, progress.DoneCount,
		), nil
	}
	return fmt.Sprintf("%s/%d", progress.Unit, progress.DoneCount), nil
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

func parseID(token string) (int, error) {
	val, err := strconv.Atoi(token)
	if err != nil {
		return -1, fmt.Errorf("%w: %w: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	if val < 0 {
		return -1, fmt.Errorf("%w: %w: negative id: %d", terrors.ErrParse, terrors.ErrValue, val)
	}
	return val, nil
}

func resolveDates(tokens []Token) []error {
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
	return errs
}

func dateToTextToken(dt *Token) {
	dt.Key = ""
	dt.Type = TokenText
	dt.Value = dt.Raw
}

func tokenizeLine(line string) ([]Token, []error) {
	specialFields := make(map[string]bool)
	var tokens []Token
	var errs []error
	handleTokenText := func(tokenStr string, err error) {
		if err != nil {
			errs = append(errs, err)
		}
		tokens = append(tokens, Token{Type: TokenText, Raw: tokenStr, Value: tokenStr})
	}
	if i, j, err := parsePriority(line); err == nil {
		p := line[i:j]
		tokens = append(tokens, Token{
			Type: TokenPriority, Key: "priority",
			Value: p,
			Raw:   fmt.Sprintf("(%s)", p),
		})
		line = line[j+1:]
	}
	for tokenStr := range strings.SplitSeq(line, " ") {
		tokenStr = strings.TrimSpace(tokenStr)
		if tokenStr == "" {
			continue
		}
		switch tokenStr[0] {
		case '+', '@', '#':
			if err := validateHint(tokenStr); err != nil {
				handleTokenText(tokenStr, nil)
				continue
			}
			tokens = append(tokens, Token{
				Type: TokenHint, Raw: tokenStr,
				Key: tokenStr[0:1], Value: tokenStr[1:],
			})
		case '$':
			keyValue := strings.SplitN(tokenStr[1:], "=", 2)
			if len(keyValue) != 2 {
				errs = append(errs, fmt.Errorf("%w: zero or multiple `=` were found: %s", terrors.ErrParse, tokenStr))
				handleTokenText(tokenStr, nil)
				continue
			}
			key, value := keyValue[0], keyValue[1]
			_, seenKey := specialFields[key]
			if key != "r" && seenKey {
				continue
			} else if key != "r" {
				specialFields[key] = true
			}
			switch key {
			case "id", "P":
				intVal, err := parseID(value)
				if err != nil {
					handleTokenText(tokenStr, fmt.Errorf("%w: $%s", err, key))
					continue
				}
				tokens = append(tokens, Token{
					Type: TokenID, Raw: tokenStr,
					Key: key, Value: intVal,
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
				tokens = append(tokens, Token{
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
				tokens = append(tokens, Token{
					Type: TokenDuration, Raw: tokenStr,
					Key: key, Value: duration,
				})
			case "p":
				progress, err := parseProgress(value)
				if err != nil {
					handleTokenText(tokenStr, err)
					continue
				}
				tokens = append(tokens, Token{
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

	task := &Task{ID: id, Text: &line}
	tokens, errs := tokenizeLine(line)
	for ndx := range tokens {
		token := tokens[ndx]
		switch token.Type {
		case TokenText:
			task.PText += token.Raw
		case TokenID:
			intVal := token.Value.(int)
			switch token.Key {
			case "id":
				task.EID = &intVal
			case "P":
				task.Parent = &intVal
			}
		case TokenHint:
			task.Hints = append(task.Hints, fmt.Sprintf("%s%s", token.Key, token.Value.(string)))
		case TokenPriority:
			task.Priority = token.Value.(string)
		case TokenDate:
			switch token.Key {
			case "c":
				val := token.Value.(*time.Time)
				if val.After(rightNow) {
					dateToTextToken(&tokens[ndx])
					continue
				}
				task.CreationDate = val
			case "lud":
				val := token.Value.(*time.Time)
				if val.After(rightNow) {
					dateToTextToken(&tokens[ndx])
					continue
				}
				task.LastUpdated = val
			case "due":
				task.DueDate = token.Value.(*time.Time)
			case "r":
				task.Reminders = append(task.Reminders, *token.Value.(*time.Time))
			case "end":
				task.EndDate = token.Value.(*time.Time)
			case "dead":
				task.Deadline = token.Value.(*time.Time)
			}
		case TokenDuration:
			task.Every = token.Value.(*time.Duration)
		case TokenProgress:
			task.Progress = *token.Value.(*Progress)
		}
	}
	task.Tokens = tokens
	if task.Temporal.CreationDate == nil {
		task.Temporal.CreationDate = &rightNow
		task.Tokens = append(task.Tokens, Token{
			Type: TokenDate, Raw: fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(rightNow)),
			Key: "c", Value: &rightNow,
		})
	}
	if task.Temporal.LastUpdated == nil {
		task.Temporal.LastUpdated = &rightNow
		ludVal := rightNow.Add(time.Second)
		task.Tokens = append(task.Tokens, Token{
			Type: TokenDate, Raw: "$lud=" + unparseDuration(time.Duration(0)),
			Key: "lud", Value: &ludVal,
		})
	}
	findToken := func(tipe TokenType, key string) (*Token, int) {
		for ndx := range task.Tokens {
			if task.Tokens[ndx].Type == tipe && task.Tokens[ndx].Key == key {
				return &task.Tokens[ndx], ndx
			}
		}
		return nil, -1
	}
	if task.DueDate != nil && !task.DueDate.After(*task.CreationDate) {
		tk, _ := findToken(TokenDate, "due")
		if tk != nil {
			dateToTextToken(tk)
			task.DueDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: due date token", terrors.ErrNotFound))
		}
	}
	if task.Deadline != nil && (task.DueDate == nil || !task.Deadline.After(*task.DueDate)) {
		tk, _ := findToken(TokenDate, "dead")
		if tk != nil {
			dateToTextToken(tk)
			task.Deadline = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
	}
	if task.EndDate != nil && (task.DueDate == nil || !task.EndDate.After(*task.DueDate)) {
		tk, _ := findToken(TokenDate, "end")
		if tk != nil {
			dateToTextToken(tk)
			task.EndDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	if task.EndDate != nil && task.Deadline != nil {
		tk, _ := findToken(TokenDate, "dead")
		if tk != nil {
			dateToTextToken(tk)
			task.Deadline = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: dead date token", terrors.ErrNotFound))
		}
		tk, _ = findToken(TokenDate, "end")
		if tk != nil {
			dateToTextToken(tk)
			task.EndDate = nil
		} else {
			errs = append(errs, fmt.Errorf("%w: end date token", terrors.ErrNotFound))
		}
	}
	if !task.LastUpdated.After(*task.CreationDate) {
		tk, _ := findToken(TokenDate, "lud")
		if tk != nil {
			tk.Raw = "$lud=" + unparseDuration(time.Duration(0))
			ludVal := task.CreationDate.Add(time.Second)
			tk.Value = &ludVal
			task.LastUpdated = &ludVal
		} else {
			errs = append(errs, fmt.Errorf("%w: lud date token", terrors.ErrNotFound))
		}
	}
	for ndx := len(task.Reminders) - 1; ndx >= 0; ndx-- {
		if !task.Reminders[ndx].After(*task.CreationDate) {
			for ndxTk := range task.Tokens {
				if task.Tokens[ndxTk].Type == TokenDate &&
					strings.HasPrefix(task.Tokens[ndxTk].Key, "r") &&
					*task.Tokens[ndxTk].Value.(*time.Time) == task.Reminders[ndx] {
					task.Reminders = slices.Delete(task.Reminders, ndx, ndx+1)
					dateToTextToken(&task.Tokens[ndxTk])
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
	var tasks []*Task = make([]*Task, 0)
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
	if viper.GetBool("debug") && errs != fmt.Errorf("") {
		return tasks, errs
	}
	return tasks, nil
}
