package task

import (
	"bytes"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
	"to-dotxt/pkg/utils"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDuration(t *testing.T) {
	assert := assert.New(t)
	helper := func(d int) string {
		v := time.Duration(d) * time.Second
		return formatDuration(&v)
	}
	const (
		minutes = 60
		hours   = minutes * 60
		days    = hours * 24
		weeks   = days * 7
		months  = days * 30
		years   = days * 365
	)
	t.Run("right now", func(t *testing.T) {
		s := helper(0)
		assert.Equal("rn", s)
	})
	t.Run("negative", func(t *testing.T) {
		s := helper(-10)
		assert.Equal("-10s", s)
	})
	t.Run("years >= 1.25", func(t *testing.T) {
		s := helper(1.265 * years)
		assert.Equal("1.2y", s)
		s = helper(12.882 * years)
		assert.Equal("12.8y", s)
		s = helper(5 * years)
		assert.Equal("5y", s)
	})
	t.Run("1 <= years < 1.25", func(t *testing.T) {
		s := helper(1.2375 * years)
		assert.Equal("1y2.8m", s)
		s = helper(1.164 * years)
		assert.Equal("1y1.9m", s)
		s = helper(1 * years)
		assert.Equal("1y", s)
		s = helper(1.004 * years)
		assert.Equal("1y", s)
	})
	t.Run("months >= 2", func(t *testing.T) {
		s := helper(12.166 * months)
		assert.Equal("12.1m", s)
		s = helper(12 * months)
		assert.Equal("12m", s)
		s = helper(2.5 * months)
		assert.Equal("2.5m", s)
		s = helper(2 * months)
		assert.Equal("2m", s)
	})
	t.Run("1 <= months < 2", func(t *testing.T) {
		s := helper(1.999 * months)
		assert.Equal("1m4.2w", s)
		s = helper(1.24 * months)
		assert.Equal("1m1w", s)
		s = helper(1.23 * months)
		assert.Equal("1m6d", s)
		s = helper(1.04 * months)
		assert.Equal("1m1d", s)
		s = helper(1.03 * months)
		assert.Equal("1m", s)
		s = helper(1 * months)
		assert.Equal("1m", s)
	})
	t.Run("weeks >= 1", func(t *testing.T) {
		s := helper(4.28 * weeks)
		assert.Equal("4w1d", s)
		s = helper(3.99 * weeks)
		assert.Equal("3w6d", s)
		s = helper(3.6 * weeks)
		assert.Equal("3w4d", s)
		s = helper(3 * weeks)
		assert.Equal("3w", s)
		s = helper(1.14 * weeks)
		assert.Equal("1w", s)
		s = helper(1 * weeks)
		assert.Equal("1w", s)
	})
	t.Run("days >= 2", func(t *testing.T) {
		s := helper(6.9 * days)
		assert.Equal("6d", s)
		s = helper(3 * days)
		assert.Equal("3d", s)
		s = helper(2 * days)
		assert.Equal("2d", s)
	})
	t.Run("1 <= days < 2", func(t *testing.T) {
		s := helper(1.99 * days)
		assert.Equal("1d23'", s)
		s = helper(1.5 * days)
		assert.Equal("1d12'", s)
		s = helper(1 * days)
		assert.Equal("1d", s)
	})
	t.Run("hours >= 2", func(t *testing.T) {
		s := helper(23.99 * hours)
		assert.Equal("23'59\"", s)
		s = helper(23 * hours)
		assert.Equal("23'", s)
		s = helper(2.01 * hours)
		assert.Equal("2'", s)
		s = helper(2 * hours)
		assert.Equal("2'", s)
	})
	t.Run("hours < 2", func(t *testing.T) {
		s := helper((1*hours + 59*minutes + 59))
		assert.Equal("1'59\"59s", s)
		s = helper((1*hours + 59*minutes))
		assert.Equal("1'59\"", s)
		s = helper((1*hours + 59))
		assert.Equal("1'59s", s)
		s = helper(1 * hours)
		assert.Equal("1'", s)
		s = helper((59*minutes + 59))
		assert.Equal("59\"59s", s)
		s = helper(59 * minutes)
		assert.Equal("59\"", s)
		s = helper(59)
		assert.Equal("59s", s)
		s = helper(1)
		assert.Equal("1s", s)
	})
}

func TestFormatAbsoluteDatetime(t *testing.T) {
	assert := assert.New(t)
	t.Run("nil dt", func(t *testing.T) {
		assert.Equal("", formatAbsoluteDatetime(nil, nil))
	})
	t.Run("nil relDt", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04-04")
		assert.Equal("2024-04-04T04-04", formatAbsoluteDatetime(dt, nil))
	})
	t.Run("duration eq", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04-04")
		assert.Equal("rn", formatAbsoluteDatetime(dt, dt))
	})
	t.Run("duration negative", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04")
		rel, _ := parseAbsoluteDatetime("2024-04-04T05-05")
		assert.Equal("-1'1\"", formatAbsoluteDatetime(dt, rel))
	})
	t.Run("duration positive", func(t *testing.T) {
		dt, _ := parseAbsoluteDatetime("2024-04-04T04-04")
		rel, _ := parseAbsoluteDatetime("2024-04-04T03-03")
		assert.Equal("1'1\"", formatAbsoluteDatetime(dt, rel))
	})
}

func TestFormatPriorities(t *testing.T) {
	assert := assert.New(t)
	data := strings.Split(strings.TrimSpace(`
	A                                        // 78 // single character
	AA                                       // 85 // single chain, no branching
	AAB                                      // 92 // chain of length 3
	AAC                                      // 95 // sibling of AAB (branching at depth 2)
	AB                                       // 98 // branch at depth 1
	ABA                                      // 101 // deeper branch under AB
	ABB                                      // 104 // sibling of ABA
	ABC                                      // 106 // third sibling under AB
	ABCD                                     // 107 // further depth under ABC
	ABCDE                                    // 108 // deeper still
	ABCDEA                                   // 108 // sibling under ABCDE
	XYZ                                      // 189 // entirely separate branch
	XYZA                                     // 207 // deeper under XYZ
	ZYX                                      // 234 // reverse order distinct
	LongPriorityStringThatExceedsDepthLimit  // 126 // very long single chain
	Short                                    // 162 // moderate length, no sibling
	1                                        // 42 // numeric single char
	12                                       // 51 // numeric two-char chain
	123                                      // 57 // numeric three-char chain
	1A                                       // 66 // mixed numeric/alpha
	-                                        // 9 // punctuation single char
	--                                       // 22 // punctuation two-char chain
	---                                      // 32 // punctuation three-char chain
	🔥                                       // 333 // single unicode char
	🔥A                                      // 351 // unicode + alpha
	特殊                                     // 270 // multi-byte unicode
	長い                                     // 306 // another unicode test
	AAAAAAAAAAAAAAAAAAAAAAAAAAAAA            // 88 // very long repeated char (25× “A”)`), "\n")
	var tasks []*rTask
	var hues []int
	for ndx := range data {
		h, _ := strconv.Atoi(strings.TrimSpace(strings.Split(data[ndx], "//")[1]))
		hues = append(hues, h)
		data[ndx] = data[ndx][:strings.Index(data[ndx], "//")]
		data[ndx] = strings.TrimSpace(data[ndx])
		data[ndx] = fmt.Sprintf("(%s)", data[ndx])
		task, _ := ParseTask(&ndx, data[ndx])
		tasks = append(tasks, task.Render(nil))
	}
	formatPriorities(tasks)
	for ndx, hue := range hues {
		h, _, _ := utils.HexToHSL(tasks[ndx].tokens[0].color)
		h = math.Round(h)
		assert.Equal(float64(hue), h)
	}
}

func TestFormatProgress(t *testing.T) {
	assert := assert.New(t)
	p := Progress{Unit: "unit", Category: "", Count: 14, DoneCount: 24}
	tokens := formatProgress(&p, 4, 5)
	var out strings.Builder
	for _, each := range tokens {
		out.WriteString(each.raw)
	}
	assert.Nil(nil)
	assert.Equal("  14/   24( 58%) ====>      (unit)", out.String())
}

func TestFormatListHeader(t *testing.T) {
	assert := assert.New(t)
	l := rList{path: "/todosFile", maxLen: 30}
	h := formatListHeader(&l)
	assert.Equal("> todosFile | ————————————————\n", h)
}

func TestResolvColor(t *testing.T) {
	assert := assert.New(t)
	assert.Equal(viper.GetString("print.color-default"), resolvColor(""))
	assert.Equal(viper.GetString("print.color-default"), resolvColor("randommmm"))
	assert.Equal(viper.GetString("print.progress.count"), resolvColor("print.progress.count"))
}

func TestColorizeToken(t *testing.T) {
	assert := assert.New(t)
	prevColor := viper.GetBool("color")
	viper.Set("color", true)

	t.Run("dominant", func(t *testing.T) {
		color := colorizeToken("text", "print.progress.doneCount", "print.progress.count")
		assert.Contains(color, "text")
		assert.Contains(color, viper.GetString("print.progress.count"))
	})
	t.Run("normal", func(t *testing.T) {
		color := colorizeToken("text", "print.progress.doneCount", "")
		assert.Contains(color, "text")
		assert.Contains(color, viper.GetString("print.progress.doneCount"))
	})

	viper.Set("color", prevColor)
}

func TestColorIds(t *testing.T) {
	assert := assert.New(t)
	out := colorizeIds(map[int]bool{1: true, 2: true, 3: true})
	// 1:"#E09952", 2:"#99E052", 3:"#52E099"
	assert.Equal("#E09952", out[1])
	assert.Equal("#99E052", out[2])
	assert.Equal("#52E099", out[3])
}

func TestFormatCategoryHeader(t *testing.T) {
	assert := assert.New(t)
	l := rList{idLen: 2, countLen: 2, doneCountLen: 2, maxLen: 30}
	out := formatCategoryHeader("some", &l)
	assert.Equal("                     some ————\n", out)
	out = formatCategoryHeader("", &l)
	assert.Equal("                          ————\n", out)
}

func TestRender(t *testing.T) {
	assert := assert.New(t)
	id := 1

	t.Run("normal", func(t *testing.T) {
		l := rList{path: "/tmp/file", idList: make(map[int]bool)}
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1w $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
		rtask := task.Render(&l)
		assert.Equal(1, l.idLen)
		assert.Equal(103, l.maxLen)
		assert.Equal(task, rtask.tsk, "task")
		assert.Equal(id, rtask.id, "id")
		assert.Equal("print.color-index", rtask.idColor, "idColor")
		assert.Equal("$p=unit/cat/2/15", rtask.tokens[0].token.Raw)
		assert.Equal("print.color-default", rtask.tokens[0].color)
		assert.Equal("(A)", rtask.tokens[1].raw)
		assert.Equal("print.color-default", rtask.tokens[1].color)
		assert.Equal("+prj", rtask.tokens[2].raw)
		assert.Equal("print.color-plus", rtask.tokens[2].color)
		assert.Equal("#tag", rtask.tokens[3].raw)
		assert.Equal("print.color-tag", rtask.tokens[3].color)
		assert.Equal("@at", rtask.tokens[4].raw)
		assert.Equal("print.color-at", rtask.tokens[4].color)
		assert.Equal("$due=1w", rtask.tokens[5].raw)
		assert.Equal("print.color-date-due", rtask.tokens[5].color)
		assert.Equal("$dead=1w", rtask.tokens[6].raw)
		assert.Equal("print.color-date-dead", rtask.tokens[6].color)
		assert.Equal("$r=6d", rtask.tokens[7].raw)
		assert.Equal("print.color-date-r", rtask.tokens[7].color)
		assert.Equal("$id=3", rtask.tokens[8].raw)
		assert.Equal("print.color-default", rtask.tokens[8].color)
		assert.Equal("$P=2", rtask.tokens[9].raw)
		assert.Equal("print.color-default", rtask.tokens[9].color)
		assert.Equal("text", rtask.tokens[10].raw)
		assert.Equal("print.color-default", rtask.tokens[10].color)
		assert.Equal("$r=4d", rtask.tokens[11].raw)
		assert.Equal("print.color-date-r", rtask.tokens[11].color)
		assert.Equal("$every=1m", rtask.tokens[12].raw)
		assert.Equal("print.color-every", rtask.tokens[12].color)
	})
	t.Run("after due", func(t *testing.T) {
		l := rList{path: "/tmp/file", idList: make(map[int]bool)}
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render(&l)
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)
		for _, tk := range rtask.tokens {
			assert.Equal("print.color-burnt", tk.dominantColor)
		}
	})
	t.Run("after due before end", func(t *testing.T) {
		l := rList{path: "/tmp/file", idList: make(map[int]bool)}
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $end=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render(&l)
		assert.Equal("text", rtask.tokens[10].raw)
		assert.Equal("print.color-running-event-text", rtask.tokens[10].color)
		assert.Equal("$end=1w5d", rtask.tokens[6].raw)
		assert.Equal("print.color-running-event", rtask.tokens[6].color)
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)
	})
	t.Run("after due before dead", func(t *testing.T) {
		l := rList{path: "/tmp/file", idList: make(map[int]bool)}
		task, _ := ParseTask(&id, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
		dt := rightNow.Add(-4 * 24 * 60 * 60 * time.Second)
		task.updateDate("due", &dt)
		rtask := task.Render(&l)
		assert.Equal("$dead=1w5d", rtask.tokens[6].raw)
		assert.Equal("print.color-imminent-deadline", rtask.tokens[6].color)
		assert.Equal("$due=-4d", rtask.tokens[5].raw)
		assert.Equal("print.color-burnt", rtask.tokens[5].color)
	})
}

func TestRenderList(t *testing.T) {
	assert := assert.New(t)
	path := "/tmp/file"

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/cat/223/3500")

	sm := rPrint{lists: make(map[string]*rList)}
	FileTasks[path] = append(FileTasks[path], task1)
	FileTasks[path] = append(FileTasks[path], task2)
	FileTasks[path] = append(FileTasks[path], task3)
	err := RenderList(&sm, path)
	assert.Nil(err)

	t.Run("nil metadata", func(t *testing.T) {
		assert.NotNil(RenderList(nil, path))
	})
	t.Run("id color", func(t *testing.T) {
		assert.Equal("#52E052", sm.lists[path].tasks[0].tokens[8].color)
		assert.Equal("#E09952", sm.lists[path].tasks[0].tokens[9].color)
	})
	t.Run("priority color", func(t *testing.T) {
		assert.Equal("#52E0E0", sm.lists[path].tasks[0].tokens[1].color)
	})
	t.Run("lengths", func(t *testing.T) {
		assert.Equal(105, sm.maxLen)
		assert.Equal(3, sm.idLen)
		assert.Equal(3, sm.countLen)
		assert.Equal(4, sm.doneCountLen)
	})
}

func TestStringify(t *testing.T) {
	assert := assert.New(t)

	id1 := 12
	helper := func(line string) string {
		task, _ := ParseTask(&id1, line)
		l := rList{path: "/tmp/file", idList: make(map[int]bool)}
		rtask := task.Render(&l)
		return rtask.stringify(false, 50)
	}
	out := helper("(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/12/15 text $r=-3d $every=1m")
	assert.Equal("12 12/15( 80%) =======>   (unit) (A) +prj #tag @at\n  $due=1d $dead=1w $r=22' $id=3 $P=2 text $r=-3d \n $every=1m", out)
	assert.Equal("12 ", out[:3], "id")
	assert.Equal("12/15( 80%) =======>   (unit) ", out[3:33], "progress")
	testLength := func(out string) bool {
		exceeds := false
		for _, each := range strings.Split(out, "\n") {
			if len(each) > 50 {
				exceeds = true
			}
		}
		return exceeds
	}
	t.Run("fold", func(t *testing.T) {
		out = helper("===========")
		assert.NotContains(out, "\n")
		assert.False(testLength(out))
		out = helper("=============================================================================================================================")
		assert.Contains(out, "\\")
		assert.Equal(2, strings.Count(out, "\\"))
		assert.Contains(out, "\n")
		assert.Equal(2, strings.Count(out, "\n"))
		assert.False(testLength(out))
		out = helper("one two three four five six seven eight nine ten eleven ============================================================= twelve thirteen fourteen sixteen seventeen eighteen nineteen twenty twenty-one")
		assert.Contains(out, "nine \n ten")
		assert.Contains(out, "fourteen\n  sixteen")
		assert.Contains(out, "twenty \n twenty-one")
		assert.Contains(out, "=====================================\\\n ========================")
		assert.False(testLength(out))
	})
}

func TestPrintLists(t *testing.T) {
	assert := assert.New(t)
	path := "/tmp/file"

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500")

	capture := func(maxlen, minlen int) []string {
		realStdout := os.Stdout
		defer func() { os.Stdout = realStdout }()
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		err = PrintLists([]string{path}, maxlen, minlen)
		require.Nil(t, err)
		w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		out := buf.String()
		return strings.Split(out, "\n")
	}

	t.Run("header len", func(t *testing.T) {
		FileTasks[path] = []*Task{task2}
		out := capture(80, 70)
		assert.Equal(192, len(out[0]))
		out = capture(90, 70)
		assert.Equal(192, len(out[0]))
		FileTasks[path] = []*Task{task1}
		out = capture(90, 70)
		assert.Equal(252, len(out[0]))
		out = capture(160, 70)
		assert.Equal(297, len(out[0]))
		out = capture(130, 70)
		assert.Equal(297, len(out[0]))
	})
	t.Run("line len", func(t *testing.T) {
		FileTasks[path] = []*Task{task1, task2, task3}
		out := capture(50, 10)
		assert.Equal(132, len(out[0])) // header
		assert.Equal(90, len(out[1]))  // category header

		assert.Equal(50, len(out[2]))
		assert.Equal(49, len(out[3]))
		assert.Equal(20, len(out[4]))

		assert.Equal(37, len(out[6]))
		assert.Equal(43, len(out[7]))
		assert.Equal(26, len(out[8]))

		assert.Equal(90, len(out[9])) // category header
		assert.Equal(15, len(out[10]))
	})
	t.Run("category headers", func(t *testing.T) {
		FileTasks[path] = []*Task{task1, task2, task3}
		out := capture(50, 10)
		assert.Equal(90, len(out[1]))
		assert.Equal(90, len(out[5]))
		assert.Equal(90, len(out[9]))
	})
}

func TestPrintTask(t *testing.T) {
	assert := assert.New(t)

	id1 := 0
	task1, _ := ParseTask(&id1, "(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m")
	id2 := 1
	task2, _ := ParseTask(&id2, "normal task")
	id3 := 210
	task3, _ := ParseTask(&id3, "tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500")
	path := "/tmp/file"
	FileTasks[path] = []*Task{task1, task2, task3}
	rn := rightNow.Format("2006-01-02T15-04-05")

	capture := func(id int) string {
		realStdout := os.Stdout
		defer func() { os.Stdout = realStdout }()
		r, w, err := os.Pipe()
		require.NoError(t, err)
		os.Stdout = w
		err = PrintTask(id, path)
		require.Nil(t, err)
		w.Close()
		var buf bytes.Buffer
		_, err = buf.ReadFrom(r)
		require.NoError(t, err)
		return buf.String()
	}

	out := capture(210)
	assert.Equal(fmt.Sprintf("tooooooooooooooooooooooooooooooooooooo looooooooooooooooooong $p=unit/223/3500 $c=%s $lud=1S\n", rn), out)
	out = capture(0)
	assert.Equal(fmt.Sprintf("(A) +prj #tag @at $due=1d $dead=1w $r=-2h $id=3 $P=2 $p=unit/cat/2/15 text $r=-3d $every=1m $c=%s $lud=1S\n", rn), out)
}
