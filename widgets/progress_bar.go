package widgets

import (
	"fmt"
	"math"
	"strings"
)

type progressBar struct {
	value float64
	width int
	flex  int
}

func ProgressBar(value float64) *progressBar {
	return &progressBar{
		value: value,
		width: 0,
		flex:  1,
	}
}

func (pb *progressBar) Width(width int) *progressBar {
	pb.width = width
	return pb
}

func (pb *progressBar) Flex(flex int) *progressBar {
	pb.flex = flex
	return pb
}

func (pb progressBar) Constraint() Constraint {
	return Constraint{Size: Size{Width: pb.width, Height: 1}, Flex: Flex{X: pb.flex, Y: 0}}
}

func (pb progressBar) Render(screen *Screen, pos Position, size Size) {
	if size.Width < 1 {
		return
	}

	runes := make([]rune, size.Width)
	progress := int(math.Round(float64(size.Width*8) * float64(pb.value)))
	idx := 0
	for ; idx < progress/8; idx++ {
		runes[idx] = '█'
	}
	if progress%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < int(size.Width); idx++ {
		runes[idx] = ' '
	}

	for x := 0; x < size.Width; x++ {
		screen.Cells[pos.Y][pos.X+x] = Cell{Rune: runes[x], Style: screen.Style}
	}
}

func (pb progressBar) String() string { return toString(pb) }

func (pb progressBar) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"ProgressBar(%.4f, %d, %d)\n", pb.value, pb.width, pb.flex)
}
