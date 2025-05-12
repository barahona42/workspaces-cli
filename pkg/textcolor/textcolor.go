package textcolor

import "fmt"

type Color = int

const (
	LIGHT_GRAY Color = 90
	YELLOW     Color = 33
	BLUE       Color = 34
)

func Colorize(color Color, text string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", color, text)
}
