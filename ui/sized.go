package ui

type sized struct {
	constraints Constraints
	widget      func(width X, height Y) Widget
}

func Sized(constraints Constraints, widget func(width X, height Y) Widget) Widget {
	return sized{constraints, widget}
}

func (s sized) Constraints() Constraints {
	return Constraints{Width: Constraint[X]{0, 1}, Height: Constraint[Y]{1, 1}}
}

func (g sized) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
	widget := g.widget(width, height)
	widget.Render(renderer, x, y, width, height, style)
}
