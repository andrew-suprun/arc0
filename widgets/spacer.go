package widgets

import (
	"fmt"
	"strings"
)

type Spacer struct{}

func (w Spacer) Constraint() Constraint {
	return Constraint{Size: Size{Width: 0, Height: 0}, Flex: Flex{X: 1, Y: 1}}
}

func (w Spacer) Render(screen *Screen, pos Position, size Size) {
	for y := 0; y < int(size.Height); y++ {
		for x := 0; x < size.Width; x++ {
			screen.Cells[pos.Y+y][pos.X+x] = Cell{Rune: ' ', Style: screen.Style}
		}
	}
}

func (s Spacer) String() string { return toString(s) }

func (s Spacer) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Spacer{}\n")
}
