package task

import (
	"dotxt/config"
	"dotxt/pkg/terrors"
	"dotxt/pkg/utils"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCleanupIDs(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("test")
	FileTasks[path] = make([]*Task, 0)
	for ndx := range 10 {
		task, _ := ParseTask(utils.MkPtr(ndx), fmt.Sprintf("%d", ndx))
		FileTasks[path] = append(FileTasks[path], task)
	}
	t.Run("nil", func(t *testing.T) {
		for _, ndx := range []int{0, 2, 4, 6, 7, 9} {
			FileTasks[path][ndx].ID = nil
		}
		cleanupIDs(path)
		for ndx := range 10 {
			if assert.NotNil(FileTasks[path][ndx].ID) {
				assert.Equal(ndx, *FileTasks[path][ndx].ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), FileTasks[path][ndx].Norm())
		}
	})
	t.Run("out of range", func(t *testing.T) {
		for ndx := range []int{0, 1, 4, 6, 7, 9} {
			sign := 1
			if ndx%2 == 0 {
				sign = -1
			}
			FileTasks[path][ndx].ID = utils.MkPtr(sign * 10 * ndx)
		}
		cleanupIDs(path)
		for ndx := range 10 {
			if assert.NotNil(FileTasks[path][ndx].ID) {
				assert.Equal(ndx, *FileTasks[path][ndx].ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), FileTasks[path][ndx].Norm())
		}
	})
	t.Run("duplicates", func(t *testing.T) {
		for ndx := range []int{1, 4, 6, 7, 9} {
			FileTasks[path][ndx].ID = utils.MkPtr(0)
		}
		cleanupIDs(path)
		for ndx := range 10 {
			if assert.NotNil(FileTasks[path][ndx].ID) {
				assert.Equal(ndx, *FileTasks[path][ndx].ID)
			}
			assert.Equal(fmt.Sprintf("%d", ndx), FileTasks[path][ndx].Norm())
		}
	})
	delete(FileTasks, path)
}

func TestAddTask(t *testing.T) {
	assert := assert.New(t)

	mkId := func(a int) *int {
		return &a
	}
	path, _ := parseFilepath("file")
	task, _ := ParseTask(mkId(1), "1")
	FileTasks[path] = []*Task{task}
	task, _ = ParseTask(nil, "nil")
	err := AddTask(task, path)
	require.Nil(t, err)
	assert.Len(FileTasks[path], 2)
	task1 := FileTasks[path][0]
	assert.Equal(0, *task1.ID)
	assert.Equal("nil", task1.Norm())
	task2 := FileTasks[path][1]
	assert.Equal(1, *task2.ID)
	assert.Equal("1", task2.Norm())
}

func TestAddTaskFromStr(t *testing.T) {
	assert := assert.New(t)
	mkId := func(a int) *int {
		return &a
	}
	path, _ := parseFilepath("file")
	task, _ := ParseTask(mkId(1), "1")
	FileTasks[path] = []*Task{task}
	err := AddTaskFromStr("nil", path)
	require.Nil(t, err)
	assert.Equal("nil", FileTasks[path][0].NormRegular())
}

func TestGetTaskIndexFromId(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	id, err := getTaskIndexFromId(1, path)
	require.Nil(t, err)
	assert.Equal(1, id)
	_, err = getTaskIndexFromId(5, path)
	require.NotNil(t, err)
	assert.ErrorIs(err, terrors.ErrNotFound)
	assert.ErrorContains(err, "task corresponding to id 5 not found")
}

func TestGetTaskFromId(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	task, err := getTaskFromId(1, path)
	require.Nil(t, err)
	assert.Equal("1", task.NormRegular())
}

func TestAppendToTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("2", path)
	err := AppendToTask(1, "new data +prj @at #tag $due=1w", path)
	require.Nil(t, err)
	task := FileTasks[path][1]
	dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
	assert.Equal(*dt, *task.Time.DueDate)
	assert.Len(task.Hints, 3)
	assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
		var out []string
		for _, h := range task.Hints {
			out = append(out, *h)
		}
		return out
	}())
	assert.Equal("1 new data", task.NormRegular())
	if assert.NotNil(task.ID) {
		assert.Equal(1, *task.ID)
	}
}

func TestPrependToTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("(A) 2", path)
	err := PrependToTask(1, "new data +prj @at #tag $due=1w", path)
	require.Nil(t, err)
	task := FileTasks[path][1]
	dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
	assert.Equal(*dt, *task.Time.DueDate)
	assert.Len(task.Hints, 3)
	assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
		var out []string
		for _, h := range task.Hints {
			out = append(out, *h)
		}
		return out
	}())
	assert.Equal("new data 1", task.NormRegular())
	if assert.NotNil(task.ID) {
		assert.Equal(1, *task.ID)
	}
	err = PrependToTask(2, "something new", path)
	require.Nil(t, err)
	task = FileTasks[path][2]
	assert.Equal("A", *task.Priority)
	assert.Equal("(A) something new 2", task.Norm())
}

func TestReplaceTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1 $c=2024-05-05T05-05", path)
	AddTaskFromStr("2", path)
	err := ReplaceTask(1, "new data +prj @at #tag $due=1w", path)
	require.Nil(t, err)
	task := FileTasks[path][1]
	dt, _ := parseAbsoluteDatetime("2024-05-12T05-05")
	assert.Equal(*dt, *task.Time.DueDate)
	assert.Len(task.Hints, 3)
	assert.Equal([]string{"+prj", "@at", "#tag"}, func() []string {
		var out []string
		for _, h := range task.Hints {
			out = append(out, *h)
		}
		return out
	}())
	assert.Equal("new data", task.NormRegular())
	if assert.NotNil(task.ID) {
		assert.Equal(1, *task.ID)
	}
	assert.NotContains(task.Norm(), "1 ")
	assert.Contains(task.Raw(), "$c=2024-05-05T05-05")
}

func TestDeduplicateList(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("(1) +prj @at #tag $due=1w $c=2024-05-05T05-05 $lud=2025-03-05T05-05 $p=page/books/0/1287 $every=1m", path)
	AddTaskFromStr("(1) +prj @at #tag $due=1w $c=1960-05-05T05-05 $lud=1970-03-05T05-05 $p=page/books/0/1287 $every=1m", path)
	AddTaskFromStr("3", path)
	err := DeduplicateList(path)
	require.NoError(t, err)
	assert.Len(FileTasks[path], 3)
	dt, _ := parseAbsoluteDatetime("2024-05-05T05-05")
	assert.Equal(*dt, *FileTasks[path][1].Time.CreationDate)
	assert.Equal("0", FileTasks[path][0].Norm())
	assert.Equal("3", FileTasks[path][2].Norm())
}

func TestDeprioritizeTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("no priority", path)
	AddTaskFromStr("(with) priority", path)
	t.Run("no priority", func(t *testing.T) {
		assert.Nil(FileTasks[path][0].Priority)
		err := DeprioritizeTask(0, path)
		assert.NoError(err)
		assert.Nil(FileTasks[path][0].Priority)
	})
	t.Run("priority", func(t *testing.T) {
		assert.Equal("with", *FileTasks[path][1].Priority)
		err := DeprioritizeTask(1, path)
		assert.NoError(err)
		assert.Nil(FileTasks[path][1].Priority)
		found := false
		for _, tk := range FileTasks[path][1].Tokens {
			if tk.Type == TokenPriority {
				found = true
			}
		}
		assert.False(found)
	})
}

func TestPrioritizeTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("no priority", path)
	AddTaskFromStr("(with) priority", path)
	t.Run("no prior priority", func(t *testing.T) {
		assert.Nil(FileTasks[path][0].Priority)
		err := PrioritizeTask(0, "prio", path)
		assert.NoError(err)
		assert.Equal("prio", *FileTasks[path][0].Priority)
		assert.Equal("(prio)", FileTasks[path][0].Tokens[0].Raw)
	})
	t.Run("prior priority", func(t *testing.T) {
		assert.Equal("with", *FileTasks[path][1].Priority)
		err := PrioritizeTask(1, "prio", path)
		assert.NoError(err)
		assert.Equal("prio", *FileTasks[path][1].Priority)
		assert.Equal("(prio)", FileTasks[path][1].Tokens[0].Raw)
	})
	t.Run("ineffective parentheses", func(t *testing.T) {
		err := PrioritizeTask(1, "(NewPrio)", path)
		assert.NoError(err)
		assert.Equal("NewPrio", *FileTasks[path][1].Priority)
		assert.Equal("(NewPrio)", FileTasks[path][1].Tokens[0].Raw)
	})
}

func TestGetIndexesFromIds(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	for ndx := range 5 {
		AddTaskFromStr(fmt.Sprintf("%d", ndx*10), path)
	}
	t.Run("happy path", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{2, 3, 4, 1, 0}, path)
		require.NoError(t, err)
		assert.ElementsMatch([]int{0, 1, 2, 3, 4}, ndxs)
		assert.Equal([]int{4, 3, 2, 1, 0}, ndxs)
	})
	t.Run("missing id", func(t *testing.T) {
		_, err := getIndexesFromIds([]int{9}, path)
		assert.ErrorIs(err, terrors.ErrNotFound)
		assert.ErrorContains(err, "ids not found")
		assert.ErrorContains(err, "9")
	})
	t.Run("support duplicates", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{1, 1, 2, 1, 1}, path)
		require.NoError(t, err)
		assert.ElementsMatch([]int{1, 2}, ndxs)
		assert.Equal([]int{2, 1}, ndxs)
	})
	t.Run("empty", func(t *testing.T) {
		ndxs, err := getIndexesFromIds([]int{}, path)
		require.NoError(t, err)
		assert.Empty(ndxs)
	})
}

func TestDeleteTasks(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5", path)
	err := DeleteTasks([]int{5, 2}, path)
	require.NoError(t, err)
	assert.Len(FileTasks[path], 4)
	assert.Equal(0, *FileTasks[path][0].ID)
	assert.Equal("0 $P=2", FileTasks[path][0].Norm())
	assert.Equal(1, *FileTasks[path][1].ID)
	assert.Equal("1", FileTasks[path][1].Norm())
	assert.Equal(2, *FileTasks[path][2].ID)
	assert.Equal("4 $P=2", FileTasks[path][2].Norm())
	assert.Equal(3, *FileTasks[path][3].ID)
	assert.Equal("3", FileTasks[path][3].Norm())
}

func TestDoneTask(t *testing.T) {
	assert := assert.New(t)

	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)

	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5", path)

	err = DoneTask([]int{5, 2}, path)
	require.NoError(t, err)
	assert.Len(FileTasks[path], 4)
	assert.Equal(0, *FileTasks[path][0].ID)
	assert.Equal("0 $P=2", FileTasks[path][0].Norm())
	assert.Equal(1, *FileTasks[path][1].ID)
	assert.Equal("1", FileTasks[path][1].Norm())
	assert.Equal(2, *FileTasks[path][2].ID)
	assert.Equal("4 $P=2", FileTasks[path][2].Norm())
	assert.Equal(3, *FileTasks[path][3].ID)
	assert.Equal("3", FileTasks[path][3].Norm())
	raw, err := os.ReadFile(filepath.Join(config.ConfigPath(), "todos", "_etc", "file.done"))
	require.NoError(t, err)
	tasks := strings.Split(string(raw), "\n")
	assert.True(strings.HasPrefix(tasks[0], "5"))
	assert.True(strings.HasPrefix(tasks[1], "2 $id=2"))
}

func TestMoveTask(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("src")
	FileTasks[path] = make([]*Task, 0)
	pathDst, _ := parseFilepath("dst")
	FileTasks[pathDst] = make([]*Task, 0)
	AddTaskFromStr("0", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2", path)
	AddTaskFromStr("3", pathDst)
	AddTaskFromStr("4", pathDst)
	AddTaskFromStr("5", pathDst)
	err := MoveTask(path, 1, pathDst)
	require.NoError(t, err)
	assert.Len(FileTasks[path], 2)
	assert.Equal(1, *FileTasks[path][1].ID)
	assert.Len(FileTasks[pathDst], 4)
	assert.Equal(3, *FileTasks[pathDst][3].ID)
	assert.Equal("1", FileTasks[pathDst][3].Norm())
}

func TestRevertTask(t *testing.T) {
	assert := assert.New(t)

	prevConfig := config.ConfigPath()
	defer config.SelectConfigFile(prevConfig)
	tmpDir, err := os.MkdirTemp(prevConfig, "")
	require.Nil(t, err)
	config.SelectConfigFile(tmpDir)
	mkDirs()

	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0 $P=2", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $id=2", path)
	AddTaskFromStr("3", path)
	AddTaskFromStr("4 $P=2", path)
	AddTaskFromStr("5", path)

	donePath := filepath.Join(config.ConfigPath(), "todos", "_etc", "file.done")
	err = os.WriteFile(donePath, []byte("1\n2\n3"), 0o655)
	require.NoError(t, err)

	err = RevertTask([]int{1}, path)
	require.NoError(t, err)
	assert.Len(FileTasks[path], 7)
	assert.Equal(6, *FileTasks[path][6].ID)
	assert.Equal("2", FileTasks[path][6].Norm())
}

func TestIncrementProgressCount(t *testing.T) {
	assert := assert.New(t)

	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0 $p=unit/cat/10/100", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $p=unit/cat/10/100", path)
	AddTaskFromStr("3 $p=unit/cat/10/100", path)
	AddTaskFromStr("4 $p=unit/cat/10/100", path)
	t.Run("no progress", func(t *testing.T) {
		err := IncrementProgressCount(1, path, 2)
		require.Error(t, err)
		assert.ErrorIs(err, terrors.ErrValue)
		assert.ErrorContains(err, "task '1' does not have a progress")
	})
	t.Run("positive", func(t *testing.T) {
		task := FileTasks[path][0]
		err := IncrementProgressCount(0, path, 2)
		require.NoError(t, err)
		assert.Equal(12, task.Prog.Count)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenProgress {
				found = true
				assert.Equal(12, tk.Value.(*Progress).Count)
				assert.Contains(tk.Raw, "12/100")
			}
		}
		assert.True(found)
	})
	t.Run("negative", func(t *testing.T) {
		task := FileTasks[path][2]
		err := IncrementProgressCount(2, path, -2)
		require.NoError(t, err)
		assert.Equal(8, task.Prog.Count)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenProgress {
				found = true
				assert.Equal(8, tk.Value.(*Progress).Count)
				assert.Contains(tk.Raw, "8/100")
			}
		}
		assert.True(found)
	})
	t.Run("exceed positive", func(t *testing.T) {
		task := FileTasks[path][3]
		err := IncrementProgressCount(3, path, 200)
		require.NoError(t, err)
		assert.Equal(100, task.Prog.Count)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenProgress {
				found = true
				assert.Equal(100, tk.Value.(*Progress).Count)
				assert.Contains(tk.Raw, "100/100")
			}
		}
		assert.True(found)
	})
	t.Run("exceed negative", func(t *testing.T) {
		task := FileTasks[path][4]
		err := IncrementProgressCount(4, path, -200)
		require.NoError(t, err)
		assert.Equal(0, task.Prog.Count)
		found := false
		for _, tk := range task.Tokens {
			if tk.Type == TokenProgress {
				found = true
				assert.Equal(0, tk.Value.(*Progress).Count)
				assert.Contains(tk.Raw, "unit/cat/0/100")
			}
		}
		assert.True(found)
	})
}

func TestCheckAndRecurTasks(t *testing.T) {
	assert := assert.New(t)
	path, _ := parseFilepath("file")
	FileTasks[path] = make([]*Task, 0)
	AddTaskFromStr("0 $c=2024-05-05T05-05 $due=1m $every=1y", path)
	AddTaskFromStr("1", path)
	AddTaskFromStr("2 $c=2024-05-05T05-05 $every=1y", path)
	AddTaskFromStr("3 $due=1w $every=1y", path)
	AddTaskFromStr("4 $due=1w $dead=1m $every=1y", path)
	AddTaskFromStr("5 $due=1w $end=1m $every=1y", path)
	AddTaskFromStr("6 $c=2024-05-05T05-05 $due=1m $dead=1m $every=1y", path)
	err := CheckAndRecurTasks(path)
	require.NoError(t, err)
	t.Run("valid", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-06-04T05-05")
		for dt.Before(rightNow) {
			dt = utils.MkPtr(dt.Add(365 * 24 * 60 * 60 * time.Second))
		}
		assert.Equal(*dt, *FileTasks[path][0].Time.DueDate)
		dueStr := unparseRelativeDatetime(*FileTasks[path][0].Time.DueDate, *FileTasks[path][0].Time.CreationDate)
		found := false
		for _, tk := range FileTasks[path][0].Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				found = true
				assert.Equal(fmt.Sprintf("$due=%s", dueStr), tk.Raw)
			}
		}
		assert.True(found)
	})
	t.Run("invalid", func(t *testing.T) {
		assert.Equal("1", FileTasks[path][1].Norm())
		assert.Equal("2 $every=1y", FileTasks[path][2].Norm())
		assert.Equal("3 $due=1w $every=1y", FileTasks[path][3].Norm())
		assert.Equal("4 $due=1w $dead=1m $every=1y", FileTasks[path][4].Norm())
		assert.Equal("5 $due=1w $end=1m $every=1y", FileTasks[path][5].Norm())
	})
	t.Run("valid with dead", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-06-04T05-05")
		for dt.Before(rightNow) {
			dt = utils.MkPtr(dt.Add(365 * 24 * 60 * 60 * time.Second))
		}
		assert.Equal(*dt, *FileTasks[path][6].Time.DueDate)
		dueStr := unparseRelativeDatetime(*FileTasks[path][6].Time.DueDate, *FileTasks[path][6].Time.CreationDate)
		found := false
		for _, tk := range FileTasks[path][6].Tokens {
			if tk.Type == TokenDate && tk.Key == "due" {
				found = true
				assert.Equal(fmt.Sprintf("$due=%s", dueStr), tk.Raw)
			}
		}
		assert.True(found)
		assert.Equal(dt.Add(30*24*60*60*time.Second), *FileTasks[path][6].Time.Deadline)
	})
}
