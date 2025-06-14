package task

import (
	"dotxt/pkg/terrors"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	assert := assert.New(t)
	checkToken := func(task *Task, key string, value *time.Time) {
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == key {
				assert.True(strings.HasPrefix(tk.Raw, "$"+key+"="))
				assert.Equal(*value, *tk.Value.(*time.Time))
				break
			}
		}
	}
	t.Run("non-temporal vars", func(t *testing.T) {
		task, _ := ParseTask(nil, "(A) +p #t @a $p=unit/cat/10/100 $due=1w $dead=1w $c=2024-05-05T05-05")
		newTask, _ := ParseTask(nil, "(B) +p #t @a $p=unit/cat/90/100 $due=1w $dead=1w")
		task.update(newTask)
		assert.Equal("B", task.Priority)
		assert.Equal(90, task.Count)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Exactly(*dt, *task.CreationDate)
		for _, tk := range task.Tokens {
			if tk.Type == TokenPriority {
				assert.Equal("B", tk.Value.(string))
			} else if tk.Type == TokenProgress {
				assert.Equal(90, tk.Value.(*Progress).Count)
			} else if tk.Type == TokenDate && tk.Key == "c" {
				assert.True(strings.HasPrefix(tk.Raw, "$c="))
				assert.Equal(*dt, *tk.Value.(*time.Time))
			}
		}
	})
	t.Run("due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=1w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("new due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=2w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("irrelavent new c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=1w $c=2025-03-03T03-03")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.CreationDate)
		checkToken(task, "c", dt)
	})
	t.Run("renewed lud", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$lud=2025-02-02T02-02")
		task.update(ntask)
		assert.Equal(rightNow, *task.LastUpdated)
		checkToken(task, "lud", &rightNow)
	})
	t.Run("general", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$c=2024-06-05T05-05 $due=1w $dead=1w $r=-2d")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.CreationDate)
		checkToken(task, "c", dt)
		assert.Equal(rightNow, *task.LastUpdated)
		checkToken(task, "lud", &rightNow)
		dt, _ = parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.Deadline)
		checkToken(task, "dead", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-10T05-05")
		assert.Equal(*dt, task.Reminders[0])
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key[0] == 'r' {
				assert.True(strings.HasPrefix(tk.Raw, "$r="))
				assert.Equal(*dt, *tk.Value.(*time.Time))
			}
		}
	})
	t.Run("retain ID", func(t *testing.T) {
		id := 2
		task, _ := ParseTask(&id, "task")
		nid := 3
		ntask, _ := ParseTask(&nid, "newtask")
		task.update(ntask)
		if assert.NotNil(task.ID) {
			assert.Equal(2, *task.ID)
		}
		ntask, _ = ParseTask(nil, "even newer")
		task.update(ntask)
		if assert.NotNil(task.ID) {
			assert.Equal(2, *task.ID)
		}
	})
}

func TestRenewLud(t *testing.T) {
	assert := assert.New(t)
	t.Run("past", func(t *testing.T) {
		task, _ := ParseTask(nil, "task $c=-1w")
		task.renewLud()
		assert.Exactly(rightNow, *task.LastUpdated)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "lud" {
				found = true
				assert.Exactly(rightNow, *tk.Value.(*time.Time))
			}
		}
		assert.True(found, "not found")
		assert.Contains(*task.Text, "$lud=7d")
	})
	t.Run("present", func(t *testing.T) {
		task, _ := ParseTask(nil, "task")
		task.renewLud()
		assert.Exactly(rightNow, *task.LastUpdated)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "lud" {
				found = true
				assert.Exactly(rightNow, *tk.Value.(*time.Time))
			}
		}
		assert.True(found, "not found")
		assert.Contains(*task.Text, "$lud=0S")
	})
}

func TestUpdateDate(t *testing.T) {
	assert := assert.New(t)
	t.Run("token not found", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m")
		dt := rightNow.Add(7 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("dead", &dt)
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "token date for field 'dead' not found")
	})
	t.Run("relative", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m")
		dt := rightNow.Add(7 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("due", &dt)
		require.NoError(t, err)
		assert.Equal("$due=7d", task.Norm())
		assert.Contains(*task.Text, "$due=7d")
		assert.Equal(dt, *task.DueDate)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				found = true
				assert.Equal("$due=7d", tk.Raw)
				assert.Equal(dt, *tk.Value.(*time.Time))
			}
		}
		assert.True(found)
	})
	t.Run("relative with var", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m $dead=variable=c;2m")
		dt := rightNow.Add(3 * 30 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("dead", &dt)
		require.NoError(t, err)
		assert.Equal("$due=1m $dead=variable=c;3m", task.Norm())
		assert.Contains(*task.Text, "$dead=variable=c;3m")
		assert.Equal(dt, *task.Deadline)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "dead" {
				found = true
				assert.Equal("$dead=variable=c;3m", tk.Raw)
				assert.Equal(dt, *tk.Value.(*time.Time))
			}
		}
		assert.True(found)
	})
	t.Run("absolute", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2025-05-05T05-05 $due=2025-06-05T05-05")
		dt, _ := parseAbsoluteDatetime("2025-07-05T05-05")
		err := task.updateDate("due", dt)
		require.NoError(t, err)
		assert.Equal("$due=2025-07-05T05-05-00", task.Norm())
		assert.Contains(*task.Text, "$due=2025-07-05T05-05-00")
		assert.Equal(*dt, *task.DueDate)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				found = true
				assert.Equal("$due=2025-07-05T05-05-00", tk.Raw)
				assert.Equal(*dt, *tk.Value.(*time.Time))
			}
		}
		assert.True(found)
	})
}
