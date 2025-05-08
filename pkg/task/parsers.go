package task

// TODO: heavily validate...
import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
	"to-dotxt/pkg/terrors"
	"to-dotxt/pkg/utils"
	"unicode"

	"github.com/spf13/viper"
)

func parseAbsoluteDatetime(absDt string) (*time.Time, error) {
	if strings.Count(absDt, "T") != 1 {
		return nil, fmt.Errorf("%w: datetime doesn't have 'T'", terrors.ErrParse)
	}
	var t time.Time
	var err error
	if dashCount := strings.Count(absDt, "-"); dashCount < 3 || dashCount > 4 {
		return nil, fmt.Errorf("%w: datetime doesn't satisfy 3 <= dashCount <= 4", terrors.ErrParse)
	} else if dashCount == 4 {
		t, err = time.Parse("2006-01-02T15-04-05", absDt)
	} else {
		t, err = time.Parse("2006-01-02T15-04", absDt)
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
	sign := 1
	if dur[0] == '-' {
		sign = -1
		dur = dur[1:]
	} else if dur[0] == '+' {
		dur = dur[1:]
	}

	const day = 24 * time.Hour
	var duration time.Duration
	var numStr string
	for _, char := range dur {
		if unicode.IsDigit(char) {
			numStr += string(char)
			continue
		}
		if numStr == "" {
			return nil, fmt.Errorf("%w: expected a number before %q", terrors.ErrParse, char)
		}
		num, err := strconv.Atoi(numStr)
		if err != nil {
			return nil, fmt.Errorf("%w: number conversion of %s failed", terrors.ErrParse, numStr)
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

func unparseRelativeDatetime(dt, rel time.Time) string {
	if dt == rel {
		return "1S"
	}
	return unparseDuration(dt.Sub(rel))
}

func parseTmpRelativeDatetime(field, dt string) (*temporalNode, error) {
	fallback, ok := temporalFallback[field]
	if !ok {
		return nil, fmt.Errorf("%w: %w: field '%s' not in temporalFallback map", terrors.ErrParse, terrors.ErrNotFound, field)
	}
	if strings.HasPrefix(dt, "variable=") {
		ndx := strings.Index(dt, ";")
		if ndx == -1 {
			return nil, fmt.Errorf("%w: did not find ';'", terrors.ErrParse)
		}
		fallback = dt[len("variable="):ndx]
		dt = dt[ndx+1:]
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
	countInt, err := strconv.Atoi(count)
	if err != nil {
		return nil, fmt.Errorf("%w: %w: $progress: count to int: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	return &Progress{
		Unit: unit, Category: category,
		Count: countInt, DoneCount: doneCountInt,
	}, nil
}

func unparseProgress(progress Progress) (string, error) {
	if progress.Unit == "" {
		return "", fmt.Errorf("%w: progress unit cannot be empty", terrors.ErrValue)
	}
	if progress.DoneCount == 0 {
		return "", fmt.Errorf("%w: progress doneCount cannot be empty", terrors.ErrValue)
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
	if j := strings.IndexRune(line, ')'); j > 0 &&
		strings.IndexFunc(line[1:j], unicode.IsSpace) == -1 {
		return 1, j, nil
	}
	return -1, -1, fmt.Errorf("%w: %w: priority", terrors.ErrParse, terrors.ErrNotFound)
}

func parseID(token string) (int, error) {
	val, err := strconv.Atoi(token)
	if err != nil {
		return -1, fmt.Errorf("%w: %w: %w", terrors.ErrParse, terrors.ErrValue, err)
	}
	return val, nil
}

func resolveDates(tokens []Token) []error {
	tokenKeyToNdx := make(map[string]int)
	nodes := make(map[string]*temporalNode)
	resolved := make(map[string]time.Time)
	resolved["rn"] = rightNow
	var totalDateCount int
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
			}
			nodes[token.Key] = v
		}
	}
	for len(resolved)-1 < totalDateCount {
		changed := false
		for _, n := range nodes {
			if _, ok := resolved[n.Field]; ok {
				continue
			}
			if base, ok := resolved[n.Ref]; ok {
				resolved[n.Field] = base.Add(*n.Offset)
				changed = true
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

func tokenizeLine(line string) ([]Token, []error) {
	// TODO: validate multiple tokens...
	var tokens []Token
	var errs []error
	handleTokenText := func(tokenStr string, err error) {
		if err != nil {
			errs = append(errs, err)
		}
		tokens = append(tokens, Token{Type: TokenText, Raw: tokenStr})
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
				if err != nil {
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
	for _, token := range tokens {
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
			// TODO(2025-05-03T20-00)
			switch token.Key {
			case "c":
				task.CreationDate = token.Value.(*time.Time)
			case "lud":
				task.LastUpdated = token.Value.(*time.Time)
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
		task.Tokens = append(task.Tokens, Token{
			Type: TokenDate, Raw: fmt.Sprintf("$lud=%s", unparseRelativeDatetime(rightNow, *task.Temporal.CreationDate)),
			Key: "c", Value: &rightNow,
		})
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
