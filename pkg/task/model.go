package task

import (
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"
	"to-dotxt/pkg/terrors"

	"maps"

	"github.com/spf13/viper"
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

func (t *Temporal) getField(key string) (*time.Time, error) {
	switch key {
	case "rn":
		return &rightNow, nil
	case "c":
		return t.CreationDate, nil
	case "lud":
		return t.LastUpdated, nil
	case "due":
		return t.DueDate, nil
	case "end":
		return t.EndDate, nil
	case "dead":
		return t.Deadline, nil
	}
	if key == "r" {
		return nil, fmt.Errorf("key r not supported since it's a slice of *time.Time")
	}
	return nil, fmt.Errorf("%w: key '%s' not found", terrors.ErrNotFound, key)
}

// The default fields for each temporal field used for
// formatting datetime relatively
var temporalFormatFallback = map[string]string{
	"c": "rn", "lud": "rn", "due": "rn",
	"end": "due", "dead": "due",
	"r": "rn",
}

func readTemporalFormatFallback() {
	tmp := viper.GetStringMapString("print.temporal-format")
	maps.Copy(temporalFormatFallback, tmp)
}

// The default fields for each temporal field used for
// parsing relative datetime
var temporalFallback = map[string]string{
	"rn":  "rn",
	"c":   "rn",
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

func (t *Task) update(new *Task) {
	creationDate := t.Temporal.CreationDate
	creationDateText := fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(*creationDate))
	*t = *new
	t.Temporal.CreationDate = creationDate
	for ndx := range t.Tokens {
		if t.Tokens[ndx].Type == TokenDate && t.Tokens[ndx].Key == "c" {
			t.Tokens[ndx].Raw = creationDateText
			t.Tokens[ndx].Value = creationDate
		}
	}
	t.updateLud()
}

func (t *Task) updateLud() {
	lud := t.Temporal.LastUpdated
	t.Temporal.LastUpdated = &rightNow
	ludText := fmt.Sprintf("$lud=%s", unparseRelativeDatetime(*lud, *t.Temporal.CreationDate))

	var token *Token
	for ndx := range t.Tokens {
		if t.Tokens[ndx].Type == TokenDate && t.Tokens[ndx].Key == "lud" {
			token = &t.Tokens[ndx]
			break
		}
	}
	if token == nil {
		t.Tokens = append(t.Tokens, Token{
			Type: TokenDate, Key: "lud",
			Raw: ludText, Value: &rightNow,
		})
		return
	}
	*t.Text = strings.Replace(*t.Text, token.Raw, ludText, 1)
	t.PText = strings.Replace(t.PText, token.Raw, ludText, 1)
}

// A reduced form of the raw string that represents tasks
// more rigidly used for comparison
func (t *Task) Norm() string {
	var out []string
	for _, token := range t.Tokens {
		if slices.Contains([]TokenType{
			TokenHint, TokenPriority,
			TokenProgress, TokenText,
		}, token.Type) {
			out = append(out, token.Raw)
		}
	}
	return strings.Join(out, " ")
}

// A reduced form of the raw string that represents tasks
// more rigidly via only regular texts used for comparison
func (t *Task) NormRegular() string {
	var out []string
	for _, token := range t.Tokens {
		if token.Type == TokenText {
			out = append(out, token.Raw)
		}
	}
	return strings.Join(out, " ")
}

func (t *Task) Raw() string {
	var out []string
	for _, token := range t.Tokens {
		out = append(out, token.Raw)
	}
	return strings.Join(out, " ")
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
