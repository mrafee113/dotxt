# to-dotxt

this is going to be designed in a way such that the texts and config are completely backwards compatible with that of todotxt.

# Docs

> Usage: [to-]dotxt [-fhpantvV] [-d todo_config] action [task_number] [task_description]

Options:
-c
    Color mode
-C
    Conky mode
-d CONFIG_FILE
    Use a configuration file other than one of the defaults:
        ~/.config/to-dotxt.yaml
-h
    Display a short help message; same as action "shorthelp"
-v
    version

Built-in Actions:
add "THING I NEED TO DO +project @context"
a "THING I NEED TO DO +project @context"
    Adds THING I NEED TO DO to your todo.txt file on its own line.
    Project and context notation optional.
    Quotes optional.

append ITEM# "TEXT TO APPEND"
app ITEM# "TEXT TO APPEND"
    Adds TEXT TO APPEND to the end of the task on line ITEM#.
    Quotes optional.

deduplicate
    Removes duplicate lines from todo.txt.

del ITEM# [TERM]
rm ITEM# [TERM]
    Deletes the task on line ITEM# in todo.txt.
    If TERM specified, deletes only TERM from the task.

depri ITEM#[, ITEM#, ITEM#, ...]
dp ITEM#[, ITEM#, ITEM#, ...]
    Deprioritizes (removes the priority) from the task(s)
    on line ITEM# in todo.txt.

pri ITEM# PRIORITY
p ITEM# PRIORITY
    Adds PRIORITY to task on line ITEM#.  If the task is already
    prioritized, replaces current priority with new PRIORITY.
    PRIORITY must be a letter between A and Z.

done ITEM#[, ITEM#, ITEM#, ...]
do ITEM#[, ITEM#, ITEM#, ...]
    Marks task(s) on line ITEM# as done in todo.txt.

move ITEM# DEST [SRC]
mv ITEM# DEST [SRC]
    Moves a line from source text file (SRC) to destination text file (DEST).
    Both source and destination file must be located in the directory defined
    in the configuration directory.  When SRC is not defined
    it's by default todo.txt.

prepend ITEM# "TEXT TO PREPEND"
prep ITEM# "TEXT TO PREPEND"
    Adds TEXT TO PREPEND to the beginning of the task on line ITEM#.
    Quotes optional.

replace ITEM# "UPDATED TODO"
    Replaces task on line ITEM# with UPDATED TODO.

revert ITEM#
reverts the ITEM line from DONE_FILE to TODO_FILE

-----------
list [TERM...]
ls [TERM...]
    Displays all tasks that contain TERM(s) sorted by priority with line
    numbers.  Each task must match all TERM(s) (logical AND); to display
    tasks that contain any TERM (logical OR), use
    'TERM1\|TERM2\|...' (with quotes), or TERM1\\|TERM2 (unquoted).
    Hides all tasks that contain TERM(s) preceded by a
    minus sign (i.e. -TERM).
    TERM(s) are grep-style basic regular expressions; for literal matching,
    put a single backslash before any [ ] \ $ * . ^ and enclose the entire
    TERM in single quotes, or use double backslashes and extra shell-quoting.
    If no TERM specified, lists entire todo.txt.

listall [TERM...]
lsa [TERM...]
    Displays all the lines in todo.txt AND done.txt that contain TERM(s)
    sorted by priority with line  numbers.  Hides all tasks that
    contain TERM(s) preceded by a minus sign (i.e. -TERM).  If no
    TERM specified, lists entire todo.txt AND done.txt
    concatenated and sorted.

listaddons
    Lists all added and overridden actions in the actions directory.

listcon [TERM...]
lsc [TERM...]
    Lists all the task contexts that start with the @ sign in todo.txt.
    If TERM specified, considers only tasks that contain TERM(s).

listpri [PRIORITIES] [TERM...]
lsp [PRIORITIES] [TERM...]
    Displays all tasks prioritized PRIORITIES.
    PRIORITIES can be a [concatenation of] single (A) or range (A-C).
    If no PRIORITIES specified, lists all prioritized tasks.
    If TERM specified, lists only prioritized tasks that contain TERM(s).
    Hides all tasks that contain TERM(s) preceded by a minus sign
    (i.e. -TERM).

lsn ITEM
prints only the ITEM line

lsp ITEM
if ITEM has a priority it will print it

-----
report
    Adds the number of open tasks and done tasks to report.txt.

count
counts the number of lines in a todo file
an additional -z makes it output the number of lines excluding the ones that start with 'z +backlog'
an additional -t makes it output the total number of lines including the ones from the done.txt file
-----

Add-on Actions:
copy ITEM
copies the ITEM line to clipboard

Options:
-@
    Hide context names in list output.  Use twice to show context
    names (default).
-+
    Hide project names in list output.  Use twice to show project
    names (default).
-c
    Color mode
-d CONFIG_FILE
    Use a configuration file other than one of the defaults:
        /home/francis/.todo/config
        /home/francis/todo.cfg
        /home/francis/.todo.cfg
        /home/francis/.config/todo/config
        /usr/local/bin/todo.cfg
        /etc/todo/config

-f
    Forces actions without confirmation or interactive input
-h
    Display a short help message; same as action "shorthelp"
-p
    Plain mode turns off colors
-P
    Hide priority labels in list output.  Use twice to show
    priority labels (default).
-a
    Don't auto-archive tasks automatically on completion
-A
    Auto-archive tasks automatically on completion
-n
    Don't preserve line numbers; automatically remove blank lines
    on task deletion
-N
    Preserve line numbers
-t
    Prepend the current date to a task automatically
    when it's added.
-T
    Do not prepend the current date to a task automatically
    when it's added.
-v