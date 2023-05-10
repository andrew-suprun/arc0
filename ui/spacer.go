package ui

type Spacer struct{}

func (w Spacer) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: Constraint[H]{0, 1}}
}

func (w Spacer) Render(renderer Renderer, x X, y Y, width W, height H, style Style) {
	runes := make([]rune, width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(height); i++ {
		renderer.Text(runes, x, y+Y(i), style)
	}
}
