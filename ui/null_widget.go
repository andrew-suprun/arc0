package ui

type NullWidget struct{}

func (w NullWidget) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 0}, Height: Constraint[H]{0, 0}}
}

func (w NullWidget) Render(renderer Renderer, x X, y Y, width W, height H, style Style) {
}
