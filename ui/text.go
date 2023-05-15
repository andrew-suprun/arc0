package ui

type text struct {
	runes      []rune
	constraint Constraint
}

func Text(txt string, width int, flex int) text {
	runes := []rune(txt)
	return text{runes, Constraint{Size{width, 1}, Flex{flex, 0}}}
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
		t.runes = append(t.runes, ' ')
		diff--
	}

	ctx.Device.Text(t.runes, pos.X, pos.Y, ctx.Style)
}
