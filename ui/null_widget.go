package ui

type NullWidget struct{}

func (w NullWidget) Constraints() Constraints {
	return Constraints{Width: Constraint[X]{0, 0}, Height: Constraint[Y]{0, 0}}
}

func (w NullWidget) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
}
