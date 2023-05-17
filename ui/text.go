package ui

import "fmt"

type text struct {
	runes      []rune
	constraint Constraint
	pad        rune
}

func Text(txt string) *text {
	runes := []rune(txt)
	return &text{runes, Constraint{Size{len(runes), 1}, Flex{0, 0}}, ' '}
}

func (t *text) String() string {
	return fmt.Sprintf("Text('%s').Width(%d).Flex(%d).Pad('%c')", string(t.runes), t.constraint.Width, t.constraint.X, t.pad)
}

func (t *text) Width(width int) *text {
	t.constraint.Width = width
	return t
}

func (t *text) Flex(flex int) *text {
	t.constraint.X = flex
	return t
}

func (t *text) Pad(r rune) *text {
	t.pad = r
	return t
}

func (t text) Constraint() Constraint {
	return t.constraint
}

func (t text) Render(ctx *Context, pos Position, size Size) {
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

	ctx.Device.Text(t.runes, pos.X, pos.Y, ctx.Style)
}
