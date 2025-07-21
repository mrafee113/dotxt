package task

import (
	"dotxt/pkg/terrors"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

func validateHint(token string) error {
	if strings.IndexAny(token, "#@+!?*&") != 0 || utf8.RuneCountInString(strings.TrimSpace(token)) < 2 {
		return fmt.Errorf("%w: token '%s' is not a hint", terrors.ErrValue, token)
	}
	return nil
}

func validateEmptyText(text string) error {
	if strings.TrimSpace(text) == "" {
		return terrors.ErrEmptyText
	}
	return nil
}

func validateHexColor(color string) error {
	if utf8.RuneCountInString(color) != 7 {
		return fmt.Errorf("%w: length of hex color must be '7'", terrors.ErrValue)
	}
	if color[0] != '#' {
		return fmt.Errorf("%w: hex color must start with '#'", terrors.ErrValue)
	}
	for _, char := range color[1:] {
		if !(unicode.IsDigit(char) || unicode.IsLetter(char)) {
			return fmt.Errorf("%w: hex color must only consist of letters and digits", terrors.ErrValue)
		}
	}
	return nil
}
