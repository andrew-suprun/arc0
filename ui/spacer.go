package ui

type Spacer struct{}

func (w Spacer) Constraints() Constraints {
	return Constraints{Width: Constraint[X]{0, 1}, Height: Constraint[Y]{0, 1}}
}

func (w Spacer) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
	runes := make([]rune, width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(height); i++ {
		renderer.Text(runes, x, y+Y(i), style)
	}
}
