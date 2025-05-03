package task

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

func (t *Task) parseVariableDuration(relVar, dur string) (*time.Time, error) {
	if strings.HasPrefix(dur, "variable=") {
		ndx := strings.Index(dur, ";")
		if ndx == -1 {
			return nil, fmt.Errorf("%w: did not find semi-colon", terrors.ErrParse)
		}
		relVar = dur[9:ndx]
		dur = dur[ndx+1:]
	}
	duration, err := parseDuration(dur)
	if err != nil {
		return nil, err
	}
	var dt *time.Time
	switch relVar {
	case "c":
		dt = t.CreationDate
	case "lud":
		dt = t.LastUpdated
	case "due":
		dt = t.DueDate
	case "end":
		dt = t.EndDate
	case "dead":
		dt = t.Deadline
	}
	if dt == nil && t.CreationDate != nil {
		dt = t.CreationDate
	} else if dt == nil {
		return nil, fmt.Errorf("%w: null date error", terrors.ErrParse)
	}
	tmp, dt := dt, new(time.Time)
	*dt = *tmp
	*dt = dt.Add(*duration)
	return dt, nil
}

func parseDate(task *Task, token, relVar string) (*time.Time, error) {
	dt, err := parseAbsoluteDatetime(token)
	if err != nil {
		dt, err = task.parseVariableDuration(relVar, token)
	}
	if err != nil {
		return nil, err
	}
	return dt, nil
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

func ParseTask(id *int, line string) (*Task, error) {
	if err := validateEmptyText(line); err != nil {
		return nil, err
	}
	task := &Task{ID: id, Text: &line}
	var PTextArr []string
	if i, j, err := parsePriority(line); err == nil {
		task.Priority = line[i:j]
	}
	tokens := strings.SplitSeq(line, " ")
	var errs []error
	handleErr := func(token string, err error) {
		errs = append(errs, err)
		PTextArr = append(PTextArr, token)
	}
	for token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		switch token[0] {
		case '+', '@', '#':
			PTextArr = append(PTextArr, token)
			if err := validateHint(token); err != nil {
				continue
			}
			task.Hints = append(task.Hints, token)
		case '$':
			keyValue := strings.SplitN(token[1:], "=", 2)
			if len(keyValue) != 2 {
				errs = append(errs, fmt.Errorf("%w: zero or multiple `=` were found: %s", terrors.ErrParse, token))
				PTextArr = append(PTextArr, token)
				continue
			}
			key, value := keyValue[0], keyValue[1]
			switch key {
			case "id":
				intVal, err := parseID(value)
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $id", err))
					continue
				}
				task.EID = &intVal
			case "P":
				intVal, err := parseID(value)
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $P", err))
					continue
				}
				task.Parent = &intVal
			case "c":
				var err error
				task.CreationDate, err = parseAbsoluteDatetime(value)
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $creationDate", err))
					continue
				}
			case "lud":
				var err error
				task.LastUpdated, err = parseDate(task, value, "c")
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $lastUpdated", err))
					continue
				}
			case "due":
				var err error
				task.DueDate, err = parseDate(task, value, "c")
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $dueDate", err))
					continue
				}
			case "end":
				var err error
				task.EndDate, err = parseDate(task, value, "due")
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $endDate", err))
					continue
				}
			case "dead":
				var err error
				task.Deadline, err = parseDate(task, value, "due")
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $deadline", err))
					continue
				}
			case "every":
				var err error
				task.Every, err = parseDuration(value)
				if err != nil {
					errs = append(errs, fmt.Errorf("%w: every: %w", terrors.ErrParse, err))
					PTextArr = append(PTextArr, token)
					continue
				}
			case "r":
				reminder, err := parseDate(task, value, "due")
				if err != nil {
					handleErr(token, fmt.Errorf("%w: $reminder", err))
					continue
				}
				task.Reminders = append(task.Reminders, *reminder)
			case "p":
				progress, err := parseProgress(value)
				if err != nil {
					handleErr(token, err)
				}
				if progress != nil {
					task.Progress = *progress
				}
			}
		default:
			PTextArr = append(PTextArr, token)
			continue
		}
	}
	if viper.GetBool("debug") {
		for _, err := range errs {
			fmt.Fprintln(os.Stderr, err)
		}
	}
	task.PText = strings.Join(PTextArr, " ")
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
