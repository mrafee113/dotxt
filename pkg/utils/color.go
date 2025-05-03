package utils

import (
	"fmt"
)

func Colorize(colorize bool, color, text string) string {
	if colorize {
		return fmt.Sprintf("${color %s}%s", color, text)
	}
	return text
}
