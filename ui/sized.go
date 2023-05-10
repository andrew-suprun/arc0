package ui

type sized struct {
	constraints Constraints
	widget      func(width W, height H) Widget
}

func Sized(constraints Constraints, widget func(width W, height H) Widget) Widget {
	return sized{constraints, widget}
}

func (s sized) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: Constraint[H]{1, 1}}
}

func (g sized) Render(renderer Renderer, x X, y Y, width W, height H, style Style) {
	widget := g.widget(width, height)
	widget.Render(renderer, x, y, width, height, style)
}
