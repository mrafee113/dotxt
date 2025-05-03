package terrors

import (
	"errors"
	"fmt"
)

var (
	ErrArgs            = errors.New("argument error")
	ErrNoArgsProvided  = fmt.Errorf("%w: no args provided", ErrArgs)
	ErrEmptyText       = errors.New("empty text error")
	ErrParse           = errors.New("failed to parse")
	ErrValue           = errors.New("value error")
	ErrListNotInMemory = errors.New("list not in memory")
	ErrNotFound        = errors.New("not found error")
)

func NewArgNotProvidedError(name string) error {
	return fmt.Errorf("%w: arg %s not provided", ErrArgs, name)
}
