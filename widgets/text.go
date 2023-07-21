package widgets

import (
	"fmt"
	"strings"
)

type text struct {
	runes []rune
	width int
	flex  int
	pad   rune
}

func Text(txt string) *text {
	runes := []rune(txt)
	return &text{runes, len(runes), 0, ' '}
}

func (t *text) Width(width int) *text {
	t.width = width
	return t
}

func (t *text) Flex(flex int) *text {
	t.flex = flex
	return t
}

func (t *text) Pad(r rune) *text {
	t.pad = r
	return t
}

func (t *text) Constraint() Constraint {
	return Constraint{Size: Size{Width: t.width, Height: 1}, Flex: Flex{X: t.flex, Y: 0}}
}

func (t *text) Render(screen *Screen, pos Position, size Size) {
	if size.Width < 1 {
		return
	}
	if len(t.runes) > int(size.Width) {
		t.runes = append(t.runes[:size.Width-1], '…')
	}
	diff := int(size.Width) - len(t.runes)
	for diff > 0 {
		t.runes = append(t.runes, t.pad)
		diff--
	}

	for x := 0; x < size.Width; x++ {
		screen.Cells[pos.Y][pos.X+x] = Cell{Rune: t.runes[x], Style: screen.Style}
	}
}

func (t *text) String() string {
	return fmt.Sprintf("Text('%s').Width(%d).Flex(%d).Pad('%c')", string(t.runes), t.width, t.flex, t.pad)
}

func (t text) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, "%s%s\n", offset, t.String())
}
