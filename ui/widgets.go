package ui

type Widget interface {
	Constraints() Constraints
	Render(renderer Renderer, x X, y Y, width W, height H, style Style)
}
