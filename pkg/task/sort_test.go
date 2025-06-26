package task

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSortTask(t *testing.T) {
	assert := assert.New(t)
	var arr []*Task
	prep := func(lines []string) {
		arr = make([]*Task, len(lines))
		for ndx, line := range lines {
			task, err := ParseTask(&ndx, line)
			assert.NoError(err)
			assert.NotNil(task)
			arr[ndx] = task
		}
		arr = sortTasks(arr)
	}
	t.Run("doneCount", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"$p=unit/0/100", "random",
		}, {
			"random", "$p=unit/0/100",
		}} {
			prep(lineSet)
			assert.Equal("$p=unit/0/100", arr[0].Norm())
			assert.Equal("random", arr[1].Norm())
		}
	})
	t.Run("category", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"$p=unit/0/100/cat", "$p=unit/0/100",
			"$p=unit/0/100/catA", "$p=unit/0/100/catB",
		}, {
			"$p=unit/0/100", "$p=unit/0/100/cat",
			"$p=unit/0/100/catB", "$p=unit/0/100/catA",
		}} {
			prep(lineSet)
			assert.Equal("$p=unit/0/100/cat", arr[0].Norm())
			assert.Equal("$p=unit/0/100", arr[3].Norm())
			assert.Equal("$p=unit/0/100/catA", arr[1].Norm())
			assert.Equal("$p=unit/0/100/catB", arr[2].Norm())
		}
	})
	t.Run("priority", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"(A) s", "s",
			"(B) s", "(C) s",
		}, {
			"s", "(A) s",
			"(C) s", "(B) s",
		}} {
			prep(lineSet)
			assert.Equal("(A) s", arr[0].Norm())
			assert.Equal("s", arr[3].Norm())
			assert.Equal("(B) s", arr[1].Norm())
			assert.Equal("(C) s", arr[2].Norm())
		}
	})
	t.Run("hints", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"+b +a", "s",
			"+b +a +z", "+z +b +0",
		}, {
			"s", "+b +a",
			"+z +b +0", "+b +a +z",
		}} {
			prep(lineSet)
			assert.Equal("+z +b +0", arr[0].Norm())
			assert.Equal("+b +a", arr[1].Norm())
			assert.Equal("+b +a +z", arr[2].Norm())
			assert.Equal("s", arr[3].Norm())
		}
	})
	t.Run("texts", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"a +b", "+b",
			"c", "d",
		}, {
			"+b", "a +b",
			"d", "c",
		}} {
			prep(lineSet)
			assert.Equal("a +b", arr[0].Norm())
			assert.Equal("+b", arr[1].Norm())
			assert.Equal("c", arr[2].Norm())
			assert.Equal("d", arr[3].Norm())
		}
	})
	t.Run("parent children", func(t *testing.T) {
		for _, lineSet := range [][]string{{
			"a",
			"b $id=1",
			"z.1 $P=1",
			"a.2 $P=1",
			"c",
		}, {
			"a.2 $P=1",
			"a",
			"z.1 $P=1",
			"c",
			"b $id=1",
		}, {
			"z.1 $P=1",
			"b $id=1",
			"c",
			"a",
			"a.2 $P=1",
		}} {
			path, _ := parseFilepath("test")
			prep := func(lines []string) {
				Lists.Empty(path)
				for ndx, line := range lines {
					task, err := ParseTask(&ndx, line)
					assert.NoError(err)
					assert.NotNil(task)
					Lists.Append(path, task)
				}
				cleanupRelations(path)
				Lists.Sort(path)
			}
			prep(lineSet)
			assert.Equal("a", Lists[path].Tasks[0].Norm())
			assert.Equal("b $id=1", Lists[path].Tasks[1].Norm())
			assert.Equal("a.2 $P=1", Lists[path].Tasks[2].Norm())
			assert.Equal("z.1 $P=1", Lists[path].Tasks[3].Norm())
			assert.Equal("c", Lists[path].Tasks[4].Norm())
		}
	})
}
