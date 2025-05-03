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
)

type Token struct {
	Type  TokenType
	Raw   string
	Key   string
	Value string
}

type Progress struct {
	Unit      string
	Category  string
	Count     int
	DoneCount int
}

type Task struct {
	Tokens   []Token
	ID       *int
	EID      *int // explicit id ($id=)
	Text     *string
	PText    string
	Hints    []string
	Priority string
	Parent   *int

	CreationDate *time.Time
	LastUpdated  *time.Time

	DueDate   *time.Time
	Reminders []time.Time
	EndDate   *time.Time
	Deadline  *time.Time
	Every     *time.Duration

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
