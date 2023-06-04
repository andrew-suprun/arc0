package widgets

import (
	"fmt"
	"strings"
)

type Spacer struct{}

func (w Spacer) Constraint() Constraint {
	return Constraint{Size: Size{Width: 0, Height: 0}, Flex: Flex{X: 1, Y: 1}}
}

func (w Spacer) Render(renderer Renderer, pos Position, size Size) {
	runes := make([]rune, size.Width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(size.Height); i++ {
		renderer.Text(runes, Position{X: pos.X, Y: pos.Y + i})
	}
}

func (s Spacer) String() string { return toString(s) }

func (s Spacer) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Spacer{}\n")
}
