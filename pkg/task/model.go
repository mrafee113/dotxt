package task

import (
	"dotxt/pkg/terrors"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"time"

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

type Temporal struct {
	CreationDate *time.Time
	LastUpdated  *time.Time
	DueDate      *time.Time
	Reminders    []*time.Time
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

func (t *Temporal) setField(key string, val *time.Time) error {
	switch key {
	case "c":
		t.CreationDate = val
	case "lud":
		t.LastUpdated = val
	case "due":
		t.DueDate = val
	case "end":
		t.EndDate = val
	case "dead":
		t.Deadline = val
	}
	if key == "r" {
		return fmt.Errorf("key r not supported since it's a slice of *time.Time")
	}
	return fmt.Errorf("%w: key '%s' not found", terrors.ErrNotFound, key)
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
	Tokens   []*Token
	ID       *int
	EID      *int // explicit id ($id=)
	Hints    []*string
	Priority *string
	Parent   *int

	Temporal
	Progress
}

func (t *Task) String() string {
	return fmt.Sprintf("%-2d %s", *t.ID, t.Raw())
}

func (t *Task) update(new *Task) error {
	creationDateText := fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(*new.Temporal.CreationDate))
	text := strings.ReplaceAll(new.Raw(), creationDateText, "")
	text = strings.ReplaceAll(text, creationDateText[:len(creationDateText)-3], "")
	creationDateText = fmt.Sprintf("$c=%s", unparseAbsoluteDatetime(*t.Temporal.CreationDate))
	new, err := ParseTask(new.ID, text+" "+creationDateText)
	if err != nil {
		return err
	}
	id := t.ID
	*t = *new
	t.ID = id
	t.renewLud()
	return nil
}

func (t *Task) updateFromText(new string) error {
	dummy, err := ParseTask(nil, new)
	if err != nil {
		return err
	}
	err = t.update(dummy)
	if err != nil {
		return err
	}
	return nil
}

func (t *Task) renewLud() {
	t.Temporal.LastUpdated = &rightNow
	ludText := fmt.Sprintf("$lud=%s", unparseRelativeDatetime(rightNow, *t.Temporal.CreationDate))

	var token *Token
	for ndx := range t.Tokens {
		if t.Tokens[ndx].Type == TokenDate && t.Tokens[ndx].Key == "lud" {
			token = t.Tokens[ndx]
			break
		}
	}
	if token == nil {
		t.Tokens = append(t.Tokens, &Token{
			Type: TokenDate, Key: "lud",
			Raw: ludText, Value: &rightNow,
		})
	} else {
		token.Value = &rightNow
		token.Raw = ludText
	}
}

func (t *Task) updateDate(field string, newDt *time.Time) error {
	var curDtTxt, newDtTxt string
	var token *Token
	for ndx := range t.Tokens {
		if t.Tokens[ndx].Type == TokenDate && t.Tokens[ndx].Key == field {
			token = t.Tokens[ndx]
			break
		}
	}
	if token == nil {
		return fmt.Errorf("%w: token date for field '%s' not found", terrors.ErrNotFound, field)
	}
	curDtTxt = strings.TrimPrefix(token.Raw, fmt.Sprintf("$%s=", field))

	_, isAbsDt := parseAbsoluteDatetime(curDtTxt)
	if isAbsDt == nil {
		newDtTxt = unparseAbsoluteDatetime(*newDt)
	} else {
		fallback, _, err := getTemporalFallback(field, curDtTxt)
		if err != nil {
			return err
		}
		rel, err := t.getField(fallback)
		if err != nil {
			return err
		}
		newDtTxt = unparseRelativeDatetime(*newDt, *rel)
		tmp := temporalFallback[field]
		if tmp != fallback {
			newDtTxt = fmt.Sprintf("variable=%s;%s", fallback, newDtTxt)
		}
	}
	token.Raw = fmt.Sprintf("$%s=%s", field, newDtTxt)
	token.Value = newDt
	t.setField(field, newDt)
	return nil
}

// A reduced form of the raw string that represents tasks
// more rigidly used for comparison
// :: everything besides $c and $lud
func (t *Task) Norm() string {
	var out []string
	for _, token := range t.Tokens {
		if token.Type == TokenDate && slices.Contains([]string{"c", "lud"}, token.Key) {
			continue
		}
		out = append(out, token.Raw)
	}
	return strings.Join(out, " ")
}

// A reduced form of the raw string that represents tasks
// more rigidly via only regular texts used for comparison
// :: only non-special text
func (t *Task) NormRegular() string {
	var out []string
	for _, token := range t.Tokens {
		if token.Type == TokenText {
			out = append(out, token.Raw)
		}
	}
	return strings.Join(out, " ")
}

// the text of the task joined in from the tokens
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
		t.ID, t.EID, t.Raw(), t.Hints, t.Priority, t.Parent, t.CreationDate, t.LastUpdated, t.DueDate, t.Reminders, t.EndDate, t.Deadline, t.Every, t.Progress)
}
