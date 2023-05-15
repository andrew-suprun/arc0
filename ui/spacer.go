package ui

type Spacer struct{}

func (w Spacer) Constraint() Constraint {
	return Constraint{Size{0, 0}, Flex{1, 1}}
}

func (w Spacer) Render(ctx *Context, pos Position, size Size) {
	runes := make([]rune, size.Width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(size.Height); i++ {
		ctx.Device.Text(runes, pos.X, pos.Y+i, ctx.Style)
	}
}
