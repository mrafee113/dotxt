package task

import (
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdate(t *testing.T) {
	assert := assert.New(t)
	checkToken := func(task *Task, key string, value *time.Time) {
		tk, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, key))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.True(strings.HasPrefix(tk.raw, "$"+key+"="))
			assert.Equal(*value, *tk.Value.(*time.Time))
		}
	}
	t.Run("non-temporal vars", func(t *testing.T) {
		task, _ := ParseTask(nil, "(A) +p #t @a $p=unit/10/100/cat $due=1w $dead=1w $c=2024-05-05T05-05")
		newTask, _ := ParseTask(nil, "(B) +p #t @a $p=unit/90/100/cat $due=1w $dead=1w")
		task.update(newTask)
		assert.Equal("B", *task.Priority)
		assert.Equal(90, task.Prog.Count)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Exactly(*dt, *task.Time.CreationDate)
		task.Tokens.ForEach(func(tk *Token) {
			if tk.Type == TokenPriority {
				assert.Equal("B", *tk.Value.(*string))
			} else if tk.Type == TokenProgress {
				assert.Equal(90, tk.Value.(*Progress).Count)
			} else if tk.Type == TokenDate && tk.Key == "c" {
				assert.True(strings.HasPrefix(tk.raw, "$c="))
				assert.Equal(*dt, *tk.Value.(*time.Time))
			}
		})
	})
	t.Run("due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=1w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("new due + old c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=2w")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
	})
	t.Run("irrelavent new c", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1w $c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$due=1w $c=2025-03-03T03-03")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.Time.CreationDate)
		checkToken(task, "c", dt)
	})
	t.Run("renewed lud", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2024-05-05T05-05 $lud=2024-06-06T06-06")
		ntask, _ := ParseTask(nil, "$lud=2025-02-02T02-02")
		task.update(ntask)
		assert.Equal(rightNow, *task.Time.LastUpdated)
		checkToken(task, "lud", &rightNow)
	})
	t.Run("general", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2024-05-05T05-05")
		ntask, _ := ParseTask(nil, "$c=2024-06-05T05-05 $due=1w $dead=1w $r=-2d")
		task.update(ntask)
		dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
		assert.Equal(*dt, *task.Time.CreationDate)
		checkToken(task, "c", dt)
		assert.Equal(rightNow, *task.Time.LastUpdated)
		checkToken(task, "lud", &rightNow)
		dt, _ = parseAbsoluteDatetime("2024-05-12T05-05")
		assert.Equal(*dt, *task.Time.DueDate)
		checkToken(task, "due", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-19T05-05")
		assert.Equal(*dt, *task.Time.Deadline)
		checkToken(task, "dead", dt)
		dt, _ = parseAbsoluteDatetime("2024-05-10T05-05")
		assert.Equal(*dt, *task.Time.Reminders[0])
		tk, ndx := task.Tokens.Find(func(tk *Token) bool {
			return tk.Type == TokenDate && tk.Key[0] == 'r'
		})
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.True(strings.HasPrefix(tk.raw, "$r="))
			assert.Equal(*dt, *tk.Value.(*time.Time))
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
		assert.Exactly(rightNow, *task.Time.LastUpdated)
		tk, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "lud"))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Exactly(rightNow, *tk.Value.(*time.Time))
			assert.Equal("$lud=1w", tk.raw)
		}
	})
	t.Run("present", func(t *testing.T) {
		task, _ := ParseTask(nil, "task")
		task.renewLud()
		assert.Exactly(rightNow, *task.Time.LastUpdated)
		tk, ndx := task.Tokens.Find(TkByTypeKey(TokenDate, "lud"))
		assert.GreaterOrEqual(ndx, 0)
		if assert.NotNil(tk) {
			assert.Exactly(rightNow, *tk.Value.(*time.Time))
			assert.Equal("$lud=0s", tk.raw)
		}
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
		assert.Equal("$due=1w", task.Norm())
		assert.Equal(dt, *task.Time.DueDate)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Equal("$due=1w", tk.raw)
			assert.Equal(dt, *tk.Value.(*time.Time))
		}
	})
	t.Run("relative with var", func(t *testing.T) {
		task, _ := ParseTask(nil, "$due=1m $dead=c:2m")
		dt := rightNow.Add(3 * 30 * 24 * 60 * 60 * time.Second)
		err := task.updateDate("dead", &dt)
		require.NoError(t, err)
		assert.Equal("$due=1m $dead=c:3m", task.Norm())
		assert.Equal(dt, *task.Time.Deadline)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "dead"))
		if assert.NotNil(tk) {
			assert.Equal("$dead=c:3m", tk.raw)
			assert.Equal(dt, *tk.Value.(*time.Time))
		}
	})
	t.Run("absolute", func(t *testing.T) {
		task, _ := ParseTask(nil, "$c=2025-05-05T05-05 $due=2025-06-05T05-05")
		dt, _ := parseAbsoluteDatetime("2025-07-05T05-05")
		err := task.updateDate("due", dt)
		require.NoError(t, err)
		assert.Equal("$due=2025-07-05T05-05", task.Norm())
		assert.Equal(*dt, *task.Time.DueDate)
		tk, _ := task.Tokens.Find(TkByTypeKey(TokenDate, "due"))
		if assert.NotNil(tk) {
			assert.Equal("$due=2025-07-05T05-05", tk.raw)
			assert.Equal(*dt, *tk.Value.(*time.Time))
		}
	})
}

func TestTokenValueTypes(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(utils.MkPtr(2), "(prio) $c=2025-05-05T05-05 $lud=1s $due=1w $dead=1w +prj #tag @at $id=1 $P=2 $every=1m $p=unit/10/100/cat")
	_, ok := task.Tokens[0].Value.(*string)
	assert.True(ok, "priority")
	_, ok = task.Tokens[1].Value.(*time.Time)
	assert.True(ok, "c date")
	_, ok = task.Tokens[2].Value.(*time.Time)
	assert.True(ok, "lud date")
	_, ok = task.Tokens[3].Value.(*time.Time)
	assert.True(ok, "duedate")
	_, ok = task.Tokens[4].Value.(*time.Time)
	assert.True(ok, "deadline")
	_, ok = task.Tokens[5].Value.(*string)
	assert.True(ok, "+hint")
	_, ok = task.Tokens[6].Value.(*string)
	assert.True(ok, "#hint")
	_, ok = task.Tokens[7].Value.(*string)
	assert.True(ok, "@hint")
	_, ok = task.Tokens[8].Value.(*string)
	assert.True(ok, "id")
	_, ok = task.Tokens[9].Value.(*string)
	assert.True(ok, "P")
	_, ok = task.Tokens[10].Value.(*time.Duration)
	assert.True(ok, "every")
	_, ok = task.Tokens[11].Value.(*Progress)
	assert.True(ok, "progress")
}

func TestLists(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("hello")
	var tasks []*Task
	for ndx := range 10 {
		task, _ := ParseTask(utils.MkPtr(ndx), fmt.Sprintf("%d", ndx))
		tasks = append(tasks, task)
	}
	t.Run("exists", func(t *testing.T) {
		Lists.Init(path)
		assert.True(Lists.Exists(path))
		Lists.Delete(path)
		assert.False(Lists.Exists(path))
	})
	t.Run("init", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		assert.True(Lists.Exists(path))
		assert.NotNil(Lists[path].EIDs)
		assert.NotNil(Lists[path].PIDs)
		assert.Empty(Lists[path].Tasks)
		Lists.Init(path, tasks...)
		assert.NotEmpty(Lists[path].Tasks)
		Lists.Delete(path)
		Lists.Init(path, tasks...)
		assert.NotEmpty(path)
	})
	t.Run("empty", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		Lists.Empty(path)
		assert.Empty(Lists[path].Tasks)
		Lists.Set(path, tasks)
		assert.NotEmpty(Lists[path].Tasks)
		Lists.Empty(path)
		assert.Empty(Lists[path].Tasks)
	})
	t.Run("set", func(t *testing.T) {
		Lists.Delete(path)
		Lists.Init(path)
		Lists.Set(path, tasks)
		assert.Equal(tasks, Lists[path].Tasks)
		Lists.Set(path, tasks[2:6])
		assert.Equal(tasks[2:6], Lists[path].Tasks)
	})
	t.Run("append", func(t *testing.T) {
		Lists.Append(path, tasks[0])
		assert.NotEqual(tasks[1], Lists[path].Tasks[len(Lists[path].Tasks)-1])
		Lists.Append(path, tasks[1])
		assert.Equal(tasks[1], Lists[path].Tasks[len(Lists[path].Tasks)-1])
	})
}

func TestString(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(nil, "(A) +prj #tag @at $due=1w $dead=1w $r=-2h $-id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m $lud=0s")
	helper := func(ndx int) string {
		return task.Tokens[ndx].String(task)
	}
	assert.Equal("(A)", helper(0))
	assert.Equal("+prj", helper(1))
	assert.Equal("#tag", helper(2))
	assert.Equal("@at", helper(3))
	assert.Equal("$due=1w", helper(4))
	assert.Equal("$dead=1w", helper(5))
	assert.Equal("$r=-2h", helper(6))
	assert.Equal("$-id=3", helper(7))
	assert.Equal("$P=2", helper(8))
	assert.Equal("$p=unit/2/15/cat", helper(9))
	assert.Equal("text", helper(10))
	assert.Equal("$r=-3d", helper(11))
	assert.Equal("$every=1m", helper(12))
	assert.Equal("$lud=0s", helper(13))
}

func TestTokens(t *testing.T) {
	assert := assert.New(t)
	task, _ := ParseTask(nil, "(A) +prj #tag @at $due=1w $dead=1w $r=-2h $id=3 $P=2 $p=unit/2/15/cat text $r=-3d $every=1m")
	keyValues := map[int]string{
		0: "(A)", 1: "+prj", 2: "#tag", 3: "@at", 4: "$due=1w",
		5: "$dead=1w", 6: "$r=-2h", 7: "$id=3", 8: "$P=2",
		9: "$p=unit/2/15/cat", 10: "text", 11: "$r=-3d",
		12: "$every=1m",
	}
	values := make([]string, len(keyValues))
	valuesKeys := make(map[string]int)
	for key, value := range keyValues {
		values[key] = value
		valuesKeys[value] = key
	}

	t.Run("simple ForEach", func(t *testing.T) {
		task.Tokens.ForEach(func(tk *Token) {
			if tk.Type == TokenDate && (tk.Key == "c" || tk.Key == "lud") {
				return
			}
			assert.Contains(values, tk.String(task))
		})
	})
	t.Run("find", func(t *testing.T) {
		tk, ndx := task.Tokens.Find(func(tk *Token) bool {
			if tk.Type == TokenID && tk.Key == "id" && *tk.Value.(*string) == "3" {
				return true
			}
			return false
		})
		assert.NotNil(tk)
		assert.Equal("$id=3", values[ndx])

		tk, ndx = task.Tokens.Find(func(tk *Token) bool {
			if tk.Type == TokenID && tk.Key == "id" && *tk.Value.(*string) == "something weird" {
				return true
			}
			return false
		})
		assert.Nil(tk)
		assert.Equal(-1, ndx)
	})
	t.Run("filter", func(t *testing.T) {
		task.Tokens.Filter(TkByType(TokenDate)).Filter(func(tk *Token) bool {
			if tk.Key == "c" || tk.Key == "lud" {
				return false
			}
			return true
		}).ForEach(func(tk *Token) {
			key := tk.String(task)
			_, ok := valuesKeys[key]
			if assert.True(ok) {
				assert.Equal(TokenDate, tk.Type)
				assert.NotEqual("c", tk.Key)
				assert.NotEqual("lud", tk.Key)
			}
		})
		task.Tokens.Filter(TkByTypeKey(TokenID, "id")).ForEach(func(tk *Token) {
			assert.Equal(TokenID, tk.Type)
			assert.Equal("id", tk.Key)
			assert.Equal("3", *tk.Value.(*string))
		})
	})
}
