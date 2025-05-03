package task

import (
	"fmt"
	"strings"
	"to-dotxt/pkg/terrors"
)

func validateHint(token string) error {
	if strings.IndexAny(token, "#@+") != 0 || len(strings.TrimSpace(token)) < 2 {
		return fmt.Errorf("%w: token %s is not a hint", terrors.ErrValue, token)
	}
	return nil
}

func validateEmptyText(text string) error {
	if strings.TrimSpace(text) == "" {
		return terrors.ErrEmptyText
	}
	return nil
}
