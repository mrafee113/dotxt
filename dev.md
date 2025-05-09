## values
* prioritize
    * what do I need to do today?
        => frontier
    * what do I need to do soon?
        => visible tasks
    * what do I need to do at some point?
        => backlogs
    * include the eisenhower matrix as well TODO
* temporal
    * calendar
        * for calendar I to need divide time-bound tasks based on how far away they are.
            for now let's say if they're later than 2 days after the end of this week (start of week is Sat),
            they're **not** gonna be **visible** in todotxt.
        * calendar representation:
            I need to print the calendar week view on the right...
            I still have to figure this part out
    * btw google calendar/tasks api is alright

    * point is to assign tasks temporally thereby reducing worries tremendously in the present
    * prevent forgetfulness & lateness
        * in the case of some task's time having had started:
            * solution for forgetfulness would be visibility of current tasks!
            * solution for lateness would be deadlines if necessary
        * in the case of some task being due in the future:
            * the solution is reminders. but the thing with reminders is that sometimes they remind you too late.
                so the task needs to be alerted contextually (which has to be updated on due time),
                especially since the point is to assign worries to timeslots and this is a potential source of worry.
                so the reminders have to be set in such a way that reminding you would also let you know it's to do
                some preparational thing for the task itself; mental or not.
* operationalize problem
    * break big tasks into little tasks
    * write tasks down
    * sometimes this process may be done when the actual task is being done rn
        => `desk` task-list
    * sometimes this process may be done (maybe even partially) at any time prior to the actual task
        => subtasks
* account for knucklehead stuff (bs tasks)
    => `small-ops`
    
## structure

* task
    - text
    - hints(+) // not required for the actual interface ; written here for the purpose of highlighting
    - hints(@)
    - hints(#)
    - priority
    - parent // the id of the task this is a subtask of ; super simple ; just one level allowed

    - creationDate // just to keep track of shit
    - lastUpdated

    - dueDate // time-bound task
    - endDate // it's an event that has an end ; depends on dueDate ; conflicts with deadline
    - deadline // it's a task but it has a deadline ; depends on dueDate ; conflicts with endDate
    - every // it's an event or a task but the dueDate, endDate and deadline will recur every `every` since dueDate

    - reminder(s) // they have to be relative to either creationDate or lastUpdated // may be added multiple times

    - unit // possible association to a category in the config ; e.g. page -> books
    - category // defaults to empty string
    - count
    - doneCount

## variable syntax
- tokens are seperated by spaces; and no escaping is allowed!!
- only the starting character makes a token special. so `#hint1#hint2` is a `hint1#hint2` # hint.
- priority must be enclosed in parentheses at the beginning of the line otherwise it's just part of the text. this is backwards compatible with todotxt;apart from the length.
- for parenting and subtasking, the parent must posess the $id variable and the child must point to the parent using the $P variable.
- absolute-datetime: %Y-%m-%dT%H-%M[-%S]
- duration: [variable=$due?5-min-prior;][-+=+][%Yy][%mm][%dd][%Hh][%MM][%SS]
- creation datetime `c`: =absolute-datetime
- last updated datetime `lud`: =absolute-datetime ; =duration `variable` defaults to `c`
- due datetime `due`: =absolute-datetime ; =duration ; =[%B][%b] ; =%Y
- end datetime `end`: =absolute-datetime ; =duration
- deadline datetime `dead`: =absolute-datetime ; =duration
- `every`: =duration use of `variable` is not allowed and it defaults to `due`
- reminder `r`: =absolute-datetime ; =duration
- progress `p`: =[:unit:]/[:category:]/[:count:]/[:doneCount:]

## printing
- each list must start with this line `> {list-name} | {list-report} ---`
- priorities should have a dynamic color system and the color system not only has to respect the ascending sort but also some depth as well
- digits lenghts should be harmonized; id, count, donecount, etc should be represented by the same amount of digits all over
- progress bar must have a color system that respects increments
- all $id= and $P= numbers should be uniquely colorized
- dates must be printed relative to other dates customized in configurations
- sorting
    - tasks with parents should be literally under the tasks with ids set; and this overrules any other sorting rule
    - tasks with progress should be sorted based on category and put at the start of list
    - tasks with priorities should be before others
    - after priorities `+` hints should be taken into account
    - after all of the above comes the regular text
- each progress category (including the empty category (unless it's just that)) should be started with a category line
- if a reminder has been passed, it should not be shown
- if a due date has been passed, it should be shown, and it should change the color of the whole task
    - unless it had an end date or a deadline that has not passed yet
        - if there was an end date and the end date has not passed the default texts should be changed to another color
        - if there was a deadline and it has not passed then only the task should be shown normally except that `$due` and `$dead` are colored differently
- develop an exponential-based color system for dates... the closer the date is to right now

## todo
- deal with `every`
- create field `$done=<absoluteDt>`
- allow parsing partial absolute datetimes, like `2025-05-07`, `2025-05`, `2025`
- develop a reporting system
- reimagine the priority coloring system
- consider progress-percentage for prioritizing tasks
- validate negative count and doneCount