package ui

type Widget interface {
	Constraints() Constraints
	Render(renderer Renderer, x X, y Y, width X, height Y, style Style)
}
