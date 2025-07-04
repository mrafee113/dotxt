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

type List struct {
	Tasks []*Task
	EIDs  map[string]*Task
	PIDs  map[*Task]string
}

type lists map[string]*List

var Lists lists = make(lists)

func (l *lists) Exists(path string) bool {
	_, ok := (*l)[path]
	return ok
}

// check existence and get
func (l *lists) Tasks(path string) ([]*Task, bool) {
	if !l.Exists(path) {
		return nil, false
	}
	return (*l)[path].Tasks, true
}

// create a list if it doesn't exist
func (l *lists) Init(path string, values ...*Task) {
	if !l.Exists(path) {
		(*l)[path] = new(List)
		(*l)[path].EIDs = make(map[string]*Task)
		(*l)[path].PIDs = make(map[*Task]string)
		if len(values) > 0 {
			(*l)[path].Tasks = append((*l)[path].Tasks, values...)
		}
	} else if len(values) > 0 {
		l.Set(path, values)
		cleanupRelations(path)
	}
}

// empties the tasks of this path if it exists
func (l *lists) Empty(path string, values ...*Task) {
	l.Init(path)
	if len(values) > 0 {
		l.Set(path, values)
	} else {
		(*l)[path].Tasks = make([]*Task, 0)
	}
}

func (l *lists) Set(path string, tasks []*Task) {
	l.Init(path)
	(*l)[path].Tasks = tasks
	cleanupRelations(path)
}

// append task to list if it exists
func (l *lists) Append(path string, task *Task) {
	l.Init(path)
	(*l)[path].Tasks = append((*l)[path].Tasks, task)
}

func (l *lists) Sort(path string) {
	if l.Exists(path) {
		(*l)[path].Tasks = sortTasks((*l)[path].Tasks)
	}
}

func (l *lists) SortFunc(path string, cmp func(*Task, *Task) int) {
	if l.Exists(path) {
		slices.SortFunc((*l)[path].Tasks, cmp)
	}
}

func (l *lists) Delete(path string) {
	if l.Exists(path) {
		delete(*l, path)
	}
}

// uses slices.Delete upon l.Tasks
func (l *lists) DeleteTasks(path string, start, end int) {
	if l.Exists(path) {
		(*l)[path].Tasks = slices.Delete((*l)[path].Tasks, start, end)
	}
}

func (l *lists) Len(path string) int {
	if l.Exists(path) {
		return len((*l)[path].Tasks)
	}
	return 0
}

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
	raw   string // an attempt to carry extra token metadata from the original text
	Key   string
	Value any // whatever the case this must be a pointer type
}

type Tokens []*Token

type TkCond func(*Token) bool
type TkFunc func(*Token)

func TkByType(tipe TokenType) TkCond {
	return func(tk *Token) bool {
		return tk.Type == tipe
	}
}

func TkByTypeKey(tipe TokenType, key string) TkCond {
	return func(tk *Token) bool {
		return tk.Type == tipe && tk.Key == key
	}
}

func (tks *Tokens) ForEach(fn TkFunc) {
	for _, tk := range *tks {
		fn(tk)
	}
}

func (tks *Tokens) Find(cond TkCond) (*Token, int) {
	for ndx, tk := range *tks {
		if cond(tk) {
			return tk, ndx
		}
	}
	return nil, -1
}

func (tks *Tokens) Filter(cond TkCond) *Tokens {
	var out Tokens
	for _, tk := range *tks {
		if cond(tk) {
			out = append(out, tk)
		}
	}
	return &out
}

// TODO: TODO TODO TODO:! REALLY IMPORTANT::: try to find a way to remove the necessity of using (task *Task). this is just wrong!
func (tk *Token) String(task *Task) string {
	switch tk.Type {
	case TokenText:
		return *tk.Value.(*string)
	case TokenID:
		val := *tk.Value.(*string)
		for _, prefix := range []string{"$id=", "$-id=", "$P="} {
			if strings.HasPrefix(tk.raw, prefix) {
				return prefix + val
			}
		}
		return val
	case TokenHint:
		return tk.Key + *tk.Value.(*string)
	case TokenPriority:
		return "(" + *tk.Value.(*string) + ")"
	case TokenDuration:
		return "$" + tk.Key + "=" + unparseDuration(*tk.Value.(*time.Duration))
	case TokenDate:
		_, err := parseAbsoluteDatetime(tk.raw)
		if err == nil {
			return unparseAbsoluteDatetime(*tk.Value.(*time.Time))
		}
		_, err = parseTmpRelativeDatetime(tk.Key, tk.raw)
		if err == nil {
			out, err := tk.unparseRelativeDatetime(task.Time, nil)
			if err == nil {
				return out
			}
		}
		return tk.raw
	case TokenProgress:
		p, err := unparseProgress(*tk.Value.(*Progress))
		if err == nil {
			return "$p=" + p
		}
		return tk.raw
	}
	return ""
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
	Tokens      Tokens
	ID          *int
	Hints       []*string
	Priority    *string
	EID         *string // explicit id ($id=)
	EIDCollapse bool
	Children    []*Task
	PID         *string // parent id ($P=)
	Parent      *Task

	Time *Temporal
	Prog *Progress
}

func (t *Task) String() string {
	return fmt.Sprintf("%-2d %s", *t.ID, t.Raw())
}

func (t *Task) Depth() int {
	count := -1
	task := t
	for task != nil {
		task = task.Parent
		count++
	}
	return count
}

func (t *Task) update(new *Task) error {
	curCreationDtToken, _ := t.Tokens.Find(TkByTypeKey(TokenDate, "c"))
	if curCreationDtToken != nil {
		newCreationDtToken, _ := new.Tokens.Find(TkByTypeKey(TokenDate, "c"))
		if newCreationDtToken != nil {
			newCreationDtToken.raw = curCreationDtToken.raw
			newCreationDtToken.Value = curCreationDtToken.Value.(*time.Time)
		} else {
			new.Tokens = append(new.Tokens, &Token{
				Type: TokenDate, Key: "c",
				raw:   curCreationDtToken.raw,
				Value: curCreationDtToken.Value.(*time.Time),
			})
		}
	}
	new, err := ParseTask(t.ID, new.Raw())
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
	dummy, err := ParseTask(t.ID, new)
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
	t.Time.LastUpdated = &rightNow
	ludText := fmt.Sprintf("$lud=%s", unparseRelativeDatetime(rightNow, *t.Time.CreationDate))

	token, _ := t.Tokens.Find(TkByTypeKey(TokenDate, "lud"))
	if token == nil {
		t.Tokens = append(t.Tokens, &Token{
			Type: TokenDate, Key: "lud",
			raw: ludText, Value: &rightNow,
		})
	} else {
		token.Value = &rightNow
		token.raw = ludText
	}
}

func (t *Task) updateDate(field string, newDt *time.Time) error {
	var curDtTxt, newDtTxt string
	token, _ := t.Tokens.Find(TkByTypeKey(TokenDate, field))
	if token == nil {
		return fmt.Errorf("%w: token date for field '%s' not found", terrors.ErrNotFound, field)
	}

	curDtTxt = strings.TrimPrefix(token.raw, fmt.Sprintf("$%s=", field))
	_, isAbsDt := parseAbsoluteDatetime(curDtTxt)
	if isAbsDt == nil {
		newDtTxt = unparseAbsoluteDatetime(*newDt)
		newDtTxt = fmt.Sprintf("$%s=%s", field, newDtTxt)
	} else {
		var err error
		newDtTxt, err = token.unparseRelativeDatetime(t.Time, newDt)
		if err != nil {
			return err
		}
	}
	token.raw = newDtTxt
	token.Value = newDt
	t.Time.setField(field, newDt)
	return nil
}

// A reduced form of the raw string that represents tasks
// more rigidly used for comparison
// :: everything besides $c and $lud
func (t *Task) Norm() string {
	var out []string
	t.Tokens.Filter(func(tk *Token) bool {
		return !(tk.Type == TokenDate && (tk.Key == "c" || tk.Key == "lud"))
	}).ForEach(func(tk *Token) {
		out = append(out, tk.String(t))
	})
	return strings.Join(out, " ")
}

// A reduced form of the raw string that represents tasks
// more rigidly via only regular texts used for comparison
// :: only non-special text
func (t *Task) NormRegular() string { // Text
	var out []string
	t.Tokens.Filter(TkByType(TokenText)).ForEach(func(tk *Token) {
		out = append(out, tk.String(t))
	})
	return strings.Join(out, " ")
}

// the text of the task joined in from the tokens
func (t *Task) Raw() string {
	var out []string
	t.Tokens.ForEach(func(tk *Token) {
		out = append(out, tk.String(t))
	})
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
		t.ID, t.EID, t.Raw(), t.Hints, t.Priority, t.PID, t.Time.CreationDate, t.Time.LastUpdated,
		t.Time.DueDate, t.Time.Reminders, t.Time.EndDate, t.Time.Deadline, t.Time.Every, t.Prog)
}
