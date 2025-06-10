package task

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"
	"to-dotxt/config"
	"to-dotxt/pkg/terrors"

	"github.com/stretchr/testify/assert"
)

const dtFormat = "2006-01-02T15-04"

func TestMain(m *testing.M) {
	config.InitViper("/tmp/dotxt-testing")
	err := os.MkdirAll("/tmp/dotxt-testing", 0755)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

func clearTasks(path string) {
	FileTasks[path] = make([]*Task, 0)
}

func mockLoad(path string) error {
	path, err := parseFilepath(path)
	clearTasks(path)
	return err
}

func TestAdd(t *testing.T) {
	assert := assert.New(t)
	path := "todo"
	mockLoad(path)
	path, err := prepFileTaskFromPath(path)
	if assert.Nil(err) {
		assert.True(strings.HasSuffix(path, "/todos/todo"))
	}

	err = AddTaskFromStr("testing", path)
	if !assert.Nil(err) {
		assert.FailNow(err.Error())
	}
	task := FileTasks[path][0]
	assert.Equal(*task.ID, 0)
	assert.Equal(*task.Text, "testing")
}

func TestParseTask(t *testing.T) {
	assert := assert.New(t)
	var found bool
	var count int
	var err error
	var task *Task
	var dt time.Time

	t.Run("weird char support", func(t *testing.T) {
		weirdChars := "`!@#$%^&**()-_=+\\/'\"[]{};:.,"
		for _, char := range weirdChars {
			task, err = ParseTask(nil, string(char))
			if assert.Nil(err, "ParseTask") {
				assert.Equalf(*task.Text, string(char), "char '%s'", string(char))
			}
		}
	})

	t.Run("validate auto creation of creationDate/lud", func(t *testing.T) {
		task, _ := ParseTask(nil, "testing")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate {
				if tk.Key == "c" {
					count++
					assert.Exactly(&rightNow, tk.Value.(*time.Time), "'c'")
				}
				if tk.Key == "lud" {
					count++
					ludVal := rightNow.Add(time.Second)
					assert.Exactly(ludVal, *tk.Value.(*time.Time), "'lud'")
				}
			}
		}
		assert.Equal(2, count, "c && lud")
	})

	t.Run("validate empty", func(t *testing.T) {
		_, err = ParseTask(nil, "")
		if !assert.NotNil(err, "ParseTask") {
			assert.FailNow(err.Error())
		}
		assert.ErrorIs(err, terrors.ErrEmptyText)
	})

	t.Run("validate priority #4", func(t *testing.T) {
		task, _ = ParseTask(nil, "(!@`#$$%^&*([]{}./';\",)")
		assert.Equal("!@`#$$%^&*([]{}./';\",", task.Priority)
	})

	t.Run("validate hints #1", func(t *testing.T) {
		task, _ = ParseTask(nil, "# + @")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenHint {
				found = true
			}
		}
		assert.False(found, "not found")
		assert.Len(task.Hints, 0)
	})
	t.Run("validate hints #2", func(t *testing.T) {
		task, _ = ParseTask(nil, "#hint +hint @hint")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenHint {
				assert.Equal("hint", tk.Value.(string))
				count++
			}
		}
		assert.Equal(3, count)
		assert.Equal([]string{"#hint", "+hint", "@hint"}, task.Hints)
	})

	t.Run("validate invalid key value: no key", func(t *testing.T) {
		task, _ = ParseTask(nil, "$")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				assert.Equal("$", tk.Value.(string))
				found = true
			}
		}
		assert.True(found, "not found")
	})
	t.Run("validate invalid key value: no equal sign", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				assert.Equal("$key", tk.Value.(string))
				found = true
			}
		}
		assert.True(found, "not found")
	})
	t.Run("validate invalid key value: no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key=")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				assert.Equal("$key=", tk.Value.(string))
				found = true
			}
		}
		assert.True(found, "not found")
	})
	t.Run("validate invalid key value: unknown key", func(t *testing.T) {
		task, _ = ParseTask(nil, "$key=value")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				assert.Equal("$key=value", tk.Value.(string))
				found = true
			}
		}
		assert.True(found, "not found")
	})
	t.Run("validate invalid key value: known key but no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				assert.Equal("$id=", tk.Value.(string))
				found = true
			}
		}
		assert.True(found, "not found")
	})

	t.Run("validate $id $P: NaN", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=noway $P=nada")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				if strings.HasPrefix(tk.Raw, "$id=") {
					count++
					assert.Equal("$id=noway", tk.Value.(string), "$id")
				}
				if strings.HasPrefix(tk.Raw, "$P=") {
					count++
					assert.Equal("$P=nada", tk.Value.(string), "$P")
				}
			}
		}
		assert.Equal(2, count, "count")
		assert.Nil(task.EID, "EID")
		assert.Nil(task.Parent, "Parent")
	})
	t.Run("validate $id $P: negative", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=-1 $P=-224")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				if strings.HasPrefix(tk.Raw, "$id=") {
					count++
					assert.Equal("$id=-1", tk.Value.(string), "$id")
				}
				if strings.HasPrefix(tk.Raw, "$P=") {
					count++
					assert.Equal("$P=-224", tk.Value.(string), "$P")
				}
			}
		}
		assert.Equal(2, count, "count")
		assert.Nil(task.EID, "EID")
		assert.Nil(task.Parent, "Parent")
	})
	t.Run("validate $id $P: valid", func(t *testing.T) {
		task, _ = ParseTask(nil, "$id=20002 $P=534")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenID {
				if strings.HasPrefix(tk.Raw, "$id=") {
					count++
					assert.Equal(20002, tk.Value.(int), "$id")
				}
				if strings.HasPrefix(tk.Raw, "$P=") {
					count++
					assert.Equal(534, tk.Value.(int), "$P")
				}
			}
		}
		assert.Equal(2, count, "count")
		if assert.NotNil(task.EID, "EID") {
			assert.Equal(20002, *task.EID, "EID")
		}
		if assert.NotNil(task.Parent, "Parent") {
			assert.Equal(534, *task.Parent, "Parent")
		}
	})

	t.Run("validate invalid absolute dates: %Y", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%d", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%d", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02T")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid absolute dates: %Y-%m-%dT%H", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=2025-04-02T20")
		assert.Nil(task.Temporal.DueDate)
	})

	t.Run("validate absolute dates", func(t *testing.T) {
		dtVal := "2025-05-20T00-00"
		dt, _ := time.ParseInLocation(dtFormat, dtVal, time.Local)
		got, err := parseAbsoluteDatetime(dtVal)
		if assert.Nil(err, "parseAbsoluteDatetime.err") {
			assert.Exactly(dt, *got, "parseAbsoluteDatetime")
		}
	})

	t.Run("validate invalid relative dates: no value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				count++
				assert.Equal("$due=", tk.Raw)
			}
		}
		assert.Equal(1, count, "count")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid relative dates: unknown unit", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=+2a")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				count++
				assert.Equal("$due=+2a", tk.Raw)
			}
		}
		assert.Equal(1, count, "count")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid relative dates: unknown relation", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=variable=abc;1y")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				count++
				assert.Equal("$due=variable=abc;1y", tk.Raw)
			}
		}
		assert.Equal(1, count, "count")
		assert.Nil(task.Temporal.DueDate)
	})
	t.Run("validate invalid relative dates: wrong syntax", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=variable=c123")
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				count++
				assert.Equal("$due=variable=c123", tk.Raw)
			}
		}
		assert.Equal(1, count, "count")
		assert.Nil(task.Temporal.DueDate)
	})

	t.Run("validate valid relative dates: relative dates", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=+1y2m3w4d5h6M7S")
		dt = rightNow.Add(38898367 * time.Second)
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				count++
				assert.Exactly(dt, *tk.Value.(*time.Time))
			}
		}
		assert.Equal(1, count, "count")
		assert.Exactly(dt, *task.DueDate)
	})
	t.Run("validate valid relative dates: valid", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=variable=lud;+1y2m3w4d5h6M7S")
		dt = rightNow.Add(38898367 * time.Second)
		count = 0
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				count++
				assert.Exactly(dt, *tk.Value.(*time.Time))
			}
		}
		assert.Equal(1, count, "count")
		assert.Exactly(dt, *task.DueDate)
	})
	t.Run("validate valid relative dates: resolve function #1", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=6m $due=1w $c=2020-01-01T01-01")
		dt, _ := parseAbsoluteDatetime("2020-01-01T01-01")
		assert.Exactly(*dt, *task.CreationDate, "CreationDate")
		assert.Exactly(dt.Add(7*24*60*60*time.Second), *task.DueDate, "DueDate")
		assert.Exactly(dt.Add((7*24*60*60+6*30*24*60*60)*time.Second), *task.Deadline, "Deadline")
	})
	t.Run("validate valid relative dates: resolve function #2", func(t *testing.T) {
		task, _ = ParseTask(nil, "$r=-2d $due=1w $end=4m")
		assert.Exactly(rightNow.Add(7*24*60*60*time.Second), *task.DueDate, "DueDate")
		assert.Exactly(rightNow.Add((7*24*60*60+4*30*24*60*60)*time.Second), *task.EndDate, "EndDate")
		assert.Contains(task.Reminders, rightNow.Add((7-2)*24*60*60*time.Second), "Reminders")
	})

	t.Run("validate date semantics: maximum count", func(t *testing.T) {
		for _, key := range []string{"c", "lud", "due", "end", "dead", "r"} {
			// if not `r`, only the first value is accepted
			// 	and any other ones are disposed of
			task, _ = ParseTask(nil, fmt.Sprintf("$%s=2026-06-06T00-00 $%s=2027-06-06T00-00", key, key))
			dt, _ := parseAbsoluteDatetime("2026-06-06T00-00")
			if key == "r" {
				assert.Equal(2, len(task.Reminders), "r count")
				assert.Contains(task.Reminders, *dt, "Reminders")
				assert.Contains(task.Reminders, dt.Add(365*24*60*60*time.Second), "Reminders")
				continue
			}
			tdt, err := task.getField(key)
			if !assert.Nil(err, key) {
				assert.Exactly(*dt, *tdt, key)
			}
		}
	})
	t.Run("validate date semantics: c maximum value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$c=1y")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "c" {
				foundDt = true
				assert.Exactly(rightNow, *tk.Value.(*time.Time))
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$c=1y", tk.Raw)
			}
		}
		assert.True(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Exactly(rightNow, *task.CreationDate, "CreationDate")
	})
	t.Run("validate date semantics: lud maximum value", func(t *testing.T) {
		task, _ = ParseTask(nil, "$lud=1y")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "lud" {
				foundDt = true
				assert.Exactly(rightNow.Add(time.Second), *tk.Value.(*time.Time))
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$lud=1y", tk.Raw)
			}
		}
		assert.True(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Exactly(rightNow.Add(time.Second), *task.LastUpdated, "LastUpdated")
	})
	t.Run("validate date semantics: dead-due existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=2026-06-06T00-00")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "dead" {
				foundDt = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$dead=2026-06-06T00-00", tk.Raw)
			}
		}
		assert.False(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Nil(task.Deadline, "Deadline") // when there's deadline but no due, deadline loses depth
	})
	t.Run("validate date semantics: end-due existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$end=1w")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "end" {
				foundDt = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$end=1w", tk.Raw)
			}
		}
		assert.False(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Nil(task.EndDate, "EndDate") // when there's end but no due, end loses depth
	})
	t.Run("validate date semantics: dead-due value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=10d $dead=variable=c;2d")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "dead" {
				foundDt = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$dead=variable=c;2d", tk.Raw)
			}
		}
		assert.False(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Nil(task.Deadline, "Deadline") // when deadline <= due, deadline loses depth
	})
	t.Run("validate date semantics: end-due value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$due=10d $end=variable=c;2d")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "end" {
				foundDt = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$end=variable=c;2d", tk.Raw)
			}
		}
		assert.False(foundDt, "found date")
		assert.True(foundTxt, "found text")
		assert.Nil(task.EndDate, "EndDate") // when end <= due, end loses depth
	})
	t.Run("validate date semantics: dead-end existence dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$dead=10d $end=1w $due=2w")
		var foundDead, foundEnd bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				if strings.Contains(tk.Raw, "dead") {
					foundDead = true
					assert.Equal("$dead=10d", tk.Raw)
				}
				if strings.Contains(tk.Raw, "end") {
					foundEnd = true
					assert.Equal("$end=1w", tk.Raw)
				}
			}
		}
		assert.True(foundDead, "found dead")
		assert.Nil(task.Deadline, "Deadline")
		assert.True(foundEnd, "found end")
		assert.Nil(task.EndDate, "EndDate") // when end & dead, then both lose depth
	})
	t.Run("validate date semantics: lud-c value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$c=2025-05-05T05-05 $lud=2025-05-04T05-05")
		assert.True(task.LastUpdated.After(*task.CreationDate))
		dt, _ := parseAbsoluteDatetime("2025-05-05T05-05")
		*dt = dt.Add(time.Second)
		assert.Exactly(*dt, *task.LastUpdated) // when lud <= c: lud=c+1S
	})
	t.Run("validate date semantics: due-c value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$c=2025-05-05T05-05 $due=2023-05-05T05-05")
		assert.Nil(task.DueDate, "DueDate")
		var foundDt, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$due=2023-05-05T05-05", tk.Raw)
			}
			if tk.Type == TokenDate && tk.Key == "due" {
				foundDt = true
			}
		}
		assert.True(foundTxt, "found text")
		assert.False(foundDt, "found date")
		assert.Nil(task.DueDate, "DueDate") // when due <= c, due loses depth
	})
	t.Run("validate date semantics: r-c value dependency", func(t *testing.T) {
		task, _ = ParseTask(nil, "$r=variable=c;-1w $r=variable=c;1w $c=2025-05-05T05-05")
		dt, _ := parseAbsoluteDatetime("2025-05-12T05-05")
		if assert.Len(task.Reminders, 1, "r count") {
			assert.Exactly(*dt, task.Reminders[0], "Reminders")
		}
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenText {
				found = true
				assert.Equal("$r=variable=c;-1w", tk.Raw)
			}
		}
		assert.True(found, "found")
	})

	t.Run("validate every", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=9y364d23h59M59S")
		val := (9*365*24*60*60 + 364*24*60*60 + 23*60*60 + 59*60 + 59) * time.Second
		assert.Exactly(val, *task.Every, "Every")
		found = false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDuration && tk.Key == "every" {
				found = true
				assert.Equal(val, *tk.Value.(*time.Duration))
			}
		}
		assert.True(found, "found")
	})
	t.Run("validate every: minimum", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=23h59M59S")
		var foundDur, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDuration && tk.Key == "every" {
				foundDur = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$every=23h59M59S", tk.Raw)
			}
		}
		assert.False(foundDur, "found duration")
		assert.True(foundTxt, "found text")
		assert.Nil(task.Every, "Every")
	})
	t.Run("validate every: maximum", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=10y")
		var foundDur, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDuration && tk.Key == "every" {
				foundDur = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$every=10y", tk.Raw)
			}
		}
		assert.False(foundDur, "found duration")
		assert.True(foundTxt, "found text")
		assert.Nil(task.Every, "Every")
	})
	t.Run("validate every: negative", func(t *testing.T) {
		task, _ = ParseTask(nil, "$every=-1w")
		var foundDur, foundTxt bool
		for _, tk := range task.Tokens {
			if tk.Type == TokenDuration && tk.Key == "every" {
				foundDur = true
			}
			if tk.Type == TokenText {
				foundTxt = true
				assert.Equal("$every=-1w", tk.Raw)
			}
		}
		assert.False(foundDur, "found duration")
		assert.True(foundTxt, "found text")
		assert.Nil(task.Every, "Every")
	})
}

func TestParsePriority(t *testing.T) {
	assert := assert.New(t)
	var err error
	t.Run("invalid: (some", func(t *testing.T) {
		_, _, err = parsePriority("(some")
		assert.ErrorIs(err, terrors.ErrNotFound, "(some")
	})
	t.Run("invalid: some)", func(t *testing.T) {
		_, _, err = parsePriority("some)")
		assert.ErrorIs(err, terrors.ErrNotFound, "some)")
		assert.ErrorContains(err, "(", "some)")
	})
	t.Run("invalid: (some )", func(t *testing.T) {
		_, _, err = parsePriority("(some )")
		assert.ErrorIs(err, terrors.ErrParse, "(some )")
		assert.ErrorContains(err, "has spaces", "(some )")
	})
	t.Run("valid: (eyo!!)", func(t *testing.T) {
		line := "(eyo!!) some things"
		i, j, err := parsePriority(line)
		assert.Nil(err, "err")
		assert.Equal(1, i, "start-index")
		assert.Equal(5+1, j, "end-index")
		assert.Equal("eyo!!", line[i:j])
	})
}

func TestParseDuration(t *testing.T) {
	assert := assert.New(t)
	var err error
	t.Run("invalid: empty", func(t *testing.T) {
		_, err = parseDuration("")
		if assert.NotNil(err) {
			assert.ErrorIs(err, terrors.ErrEmptyText)
		}
	})
	t.Run("invalid: number conversion", func(t *testing.T) {
		_, err = parseDuration("+1y*")
		if assert.NotNil(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "number conversion of ")
		}
	})
	t.Run("invalid: unexpected time unit", func(t *testing.T) {
		_, err = parseDuration("+1y2*")
		if assert.NotNil(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "unexpected time unit")
		}
	})
	t.Run("invalid: trailing numbers", func(t *testing.T) {
		_, err = parseDuration("+1y23")
		if assert.NotNil(err) {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "trailing numbers without a ")
		}
	})
	t.Run("valid: positive sign", func(t *testing.T) {
		val, err := parseDuration("+1y2m3w4d5h6M7S")
		if assert.Nil(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: no sign", func(t *testing.T) {
		val, err := parseDuration("1y2m3w4d5h6M7S")
		if assert.Nil(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: negative sign", func(t *testing.T) {
		val, err := parseDuration("-1y2m3w4d5h6M7S")
		if assert.Nil(err, "error") {
			assert.Equal(-(7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
	t.Run("valid: scrambled", func(t *testing.T) {
		val, err := parseDuration("2m4d3w7S6M1y5h")
		if assert.Nil(err, "error") {
			assert.Equal((7+6*60+5*60*60+4*24*60*60+3*7*24*60*60+2*30*24*60*60+1*365*24*60*60)*time.Second, *val)
		}
	})
}

func TestUnparseDuration(t *testing.T) {
	assert := assert.New(t)
	const (
		sec   = 1 * time.Second
		min   = 60 * sec
		hour  = min * 60
		day   = hour * 24
		month = day * 30
		year  = month * 365
	)
	t.Run("huge positive", func(t *testing.T) {
		dur := unparseDuration(6*year + 1000*month + 1000*day + 1000*hour + 1000*min + 1000*sec)
		assert.Equal("265y17d8h56M40S", dur)
	})
	t.Run("negative", func(t *testing.T) {
		dur := unparseDuration(-1 * day)
		assert.Equal("-1d", dur)
	})
}

func TestParseProgress(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: 4-parter", func(t *testing.T) {
		p, err := parseProgress("unit/cat/10/100")
		if assert.Nil(err, "err") {
			assert.Equal("unit", p.Unit, "Unit")
			assert.Equal("cat", p.Category, "Category")
			assert.Equal(10, p.Count, "Count")
			assert.Equal(100, p.DoneCount, "DoneCount")
		}
	})
	t.Run("valid: 3-parter", func(t *testing.T) {
		p, err := parseProgress("unit/10/100")
		if assert.Nil(err, "err") {
			assert.Equal("unit", p.Unit, "Unit")
			assert.Equal("", p.Category, "Category")
			assert.Equal(10, p.Count, "Count")
			assert.Equal(100, p.DoneCount, "DoneCount")
		}
	})
	t.Run("valid: 2-parter", func(t *testing.T) {
		p, err := parseProgress("unit/100")
		if assert.Nil(err, "err") {
			assert.Equal("unit", p.Unit, "Unit")
			assert.Equal("", p.Category, "Category")
			assert.Equal(0, p.Count, "Count")
			assert.Equal(100, p.DoneCount, "DoneCount")
		}
	})
	t.Run("invalid: sep count", func(t *testing.T) {
		for _, val := range []string{"unnit", "unit/cat/count/doneCount/extraValue"} {
			_, err := parseProgress(val)
			if assert.NotNil(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorContains(err, "$progress: number of `/` is either less than 2 or greater than 4")
			}
		}
	})
	t.Run("invalid: doneCount", func(t *testing.T) {
		for _, val := range []string{"unit/cat/10/!!", "unit/10/!!", "unit/!!"} {
			_, err := parseProgress(val)
			if assert.NotNil(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "$progress: doneCount to int")
				assert.ErrorContains(err, "!!")
			}
		}
	})
	t.Run("invalid: count", func(t *testing.T) {
		for _, val := range []string{"unit/cat/!!/100", "unit/!!/100"} {
			_, err := parseProgress(val)
			if assert.NotNil(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorIs(err, terrors.ErrValue)
				assert.ErrorContains(err, "$progress: count to int")
				assert.ErrorContains(err, "!!")
			}
		}
	})
	t.Run("invalid: minimum doneCount", func(t *testing.T) {
		_, err := parseProgress("unit/cat/10/-1000")
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "$progress: doneCount minimum is 1")
		}
	})
	t.Run("invalid: minimum count", func(t *testing.T) {
		_, err := parseProgress("unit/cat/-10/100")
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "$progress: count minimum is 0")
		}
	})
	t.Run("invalid: maximum count", func(t *testing.T) {
		p, err := parseProgress("unit/cat/200/100")
		assert.Nil(err, "err")
		assert.Equal(p.Count, 100) // if count is greater than or equals to doneCount, it becomes doneCount
	})
}

func TestUnparseProgress(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: 2-parter", func(t *testing.T) {
		val, err := unparseProgress(Progress{Unit: "unit", DoneCount: 100})
		assert.Nil(err, "err")
		assert.Equal("unit/100", val)
	})
	t.Run("valid: 3-parter", func(t *testing.T) {
		val, err := unparseProgress(Progress{Unit: "unit", Count: 10, DoneCount: 100})
		assert.Nil(err, "err")
		assert.Equal("unit/10/100", val)
	})
	t.Run("valid: 4-parter", func(t *testing.T) {
		val, err := unparseProgress(Progress{Unit: "unit", Category: "cat", Count: 10, DoneCount: 100})
		assert.Nil(err, "err")
		assert.Equal("unit/cat/10/100", val)
	})
	t.Run("invalid: no unit", func(t *testing.T) {
		_, err := unparseProgress(Progress{DoneCount: 100})
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "unit cannot be empty")
		}
	})
	t.Run("invalid: minimum doneCount", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit"})
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "doneCount cannot be less than 1")
		}
	})
	t.Run("invalid: minimum count", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit", Count: -1, DoneCount: 100})
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "count cannot be less than 0")
		}
	})
	t.Run("invalid: maximum count", func(t *testing.T) {
		_, err := unparseProgress(Progress{Unit: "unit", Count: 200, DoneCount: 100})
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrValue)
			assert.ErrorContains(err, "count cannot be greater than doneCount")
		}
	})
}

func TestParseAbsoluteDatetime(t *testing.T) {
	assert := assert.New(t)
	t.Run("valid: %H-%M", func(t *testing.T) {
		val, err := parseAbsoluteDatetime("2025-05-05T05-05")
		assert.Nil(err, "err")
		dt, _ := parseAbsoluteDatetime("2025-05-05T05-05")
		assert.Exactly(*dt, *val)
	})
	t.Run("valid: %H-%M-%S", func(t *testing.T) {
		val, err := parseAbsoluteDatetime("2025-05-05T05-05-05")
		assert.Nil(err, "err")
		dt, _ := parseAbsoluteDatetime("2025-05-05T05-05-05")
		assert.Exactly(*dt, *val)
	})
	t.Run("invalid: no T", func(t *testing.T) {
		_, err := parseAbsoluteDatetime("2025-05-05-05-05")
		if assert.NotNil(err, "err") {
			assert.ErrorIs(err, terrors.ErrParse)
			assert.ErrorContains(err, "datetime doesn't have 'T'")
		}
	})
	t.Run("invalid: not enough or too many dashes", func(t *testing.T) {
		for _, val := range []string{"2025-05-05T05", "2025-05T05-05", "2025T05-05-05", "-3000-05-05T05-05-05"} {
			_, err := parseAbsoluteDatetime(val)
			if assert.NotNil(err, "err") {
				assert.ErrorIs(err, terrors.ErrParse)
				assert.ErrorContains(err, "datetime doesn't satisfy 3 <= dashCount <= 4")
			}
		}
	})
}

func TestResolveDates(t *testing.T) {
	// TODO
}

func TestParseTaskS(t *testing.T) {
	// TODO
}
