package task

import (
	"fmt"
	"reflect"
	"time"
)

var FileTasks map[string][]*Task = make(map[string][]*Task)

type TokenType int

const (
	TokenText TokenType = iota
	TokenID
	TokenHint
	TokenPriority
	TokenDate
	TokenDuration
	TokenProgress
)

type Token struct {
	Type  TokenType
	Raw   string
	Key   string
	Value any
}

type Progress struct {
	Unit      string
	Category  string
	Count     int
	DoneCount int
}

// TODO(2025-05-03T20-00)
// change *time.Time to a struct which holds `Ref` for relative fields...
// that way e.g. for when updating a field or deleting it, you know
// which field is related to which field and so on and so forth...
type Temporal struct {
	CreationDate *time.Time
	LastUpdated  *time.Time
	DueDate      *time.Time
	Reminders    []time.Time
	EndDate      *time.Time
	Deadline     *time.Time
	Every        *time.Duration
}

// The default fields for each temporal field used for
// parsing relative datetime
var temporalFallback = map[string]string{
	"c":   "", // c has no fallback
	"lud": "c", "due": "c",
	"end": "due", "dead": "due", "r": "due",
}

// The temporary contrainer to hold parsed duration until offset
// datetime is resolved during parsing relative datetime
type temporalNode struct {
	Field    string
	Ref      string
	Offset   *time.Duration
	Absolute *time.Time
}

type Task struct {
	Tokens   []Token
	ID       *int
	EID      *int    // explicit id ($id=)
	Text     *string // this is the line raw text
	PText    string  // I don't know what this is, yet...
	Hints    []string
	Priority string
	Parent   *int

	Temporal
	Progress
}

func (t *Task) String() string {
	return fmt.Sprintf("%-2d %s", *t.ID, t.PText)
}

func helper[T any](p T) string {
	v := reflect.ValueOf(p)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "nil"
		}
		return fmt.Sprintf("%v", v.Elem())
	}
	return fmt.Sprintf("%v", p)
}
func print(args ...any) {
	key, _ := args[0].(string)
	var values []any
	for ndx := 1; ndx < len(args); ndx++ {
		values = append(values, helper(args[ndx]))
	}
	fmt.Printf(key, values...)
}

func DebugTask(t *Task) {
	if t == nil {
		fmt.Println("task == nil")
		return
	}
	print("id: %v, explicitId: %v\ntext: %v\nhints: %v\npriority: %v\nparent: %v\n\ncreationDate: %v\nlastUpdated: %v\n\ndueDate: %v\nreminders: %v\nendDate: %v\ndeadline: %v\nevery: %v\n\nprogress: %v\n",
		t.ID, t.EID, t.Text, t.Hints, t.Priority, t.Parent, t.CreationDate, t.LastUpdated, t.DueDate, t.Reminders, t.EndDate, t.Deadline, t.Every, t.Progress)
}
