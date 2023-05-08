package ui

type List struct {
	Header func() Widget
	Row    func(line Y) Widget
}

func (w List) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: Constraint[H]{1, 1}}
}

func (w List) Render(renderer Renderer, x X, y Y, width W, height H, style Style) {
	w.Header().Render(renderer, x, y, width, 1, style)
	for i := Y(0); i < Y(height)-1; i++ {
		w.Row(i).Render(renderer, x, y+i+1, width, 1, style)
	}
}
