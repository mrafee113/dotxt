$filter=n+5 top 5
$filter=n-5 bottom 5

text:
	- something
	- some\ thing
	- "some thing"
	- 'some "thing'
	- `some thing`

text-func:
	- has: strings.Contains
	- lt: <
	- lte: <=
	- gt: >
	- gte: >=
	- eq: ==
	- r<[text]>: regex pattern

number-func:
	- lt: <
	- lte: <=
	- gt: >
	- gte: >=
	- eq: ==

temporal-key:
	- c
	- rn
	- due
	- dead
	- end

temporal-func:
	- lt: t.Before
	- lte: t.Before || t.Equals
	- gt: t.After
	- gte: t.After || t.Equals
	- eq: t.Equals
	- is: exists or not

temporal-value:
	- true or false, for `is`
	- absolute date[time]
	- relative date[time] for which the `temporal-key` defaults to `rn`

$filter=prio[rity]:[text-func]:[text] priority
$filter=prog[ress]:[c[ount]:[number-func]:[number]][,d[done[[-]count]:[number-func]:[number]][,u:[text-func]:[text]][,C[ategory]:[text-func]:[text]] progress
$filter=date:[temporal-key]:[temporal-func]:[temporal-value]
$filter=text:[text-function]:[text] joins all text and hint tokens and evaluates it against the function and the given text
$filter=hint:[hint-key]:[text-func]:[text] hints
$filter= ? composite

--- composite
type CompositeFilterType int
const (
	CompositeFilterToken CompositeFilterType = iota
	CompositeFilterFunc
)

type FilterFunc(task *Task, tipe string, keyValue ...any) bool

type CompositeFilter struct {
	type CompostieFilterType
	token string
	function *FilterFunc
}
// this is to be treated as a stack
type Filter []*CompositeFilter

// this function would process the stack and combine the results of each function as necessary, just like a calculator but more complicated
func (f *Filter) Eval(t *Task) bool
