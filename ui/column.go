package ui

type column struct {
	constraint Constraint[H]
	widgets    []Widget
}

func Column(flex Flex, widgets ...Widget) Widget {
	height := H(0)
	for _, widget := range widgets {
		height += widget.Constraints().Height.Size
	}
	return column{Constraint[H]{height, flex}, widgets}
}

func (c column) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: c.constraint}
}

func (c column) Render(renderer Renderer, x X, y Y, width W, height H, style Style) {
	sizes := make([]Constraint[H], len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraints().Height
	}
	heights := calcSizes(height, sizes)
	for i, widget := range c.widgets {
		widget.Render(renderer, x, y, width, height, style)
		y = y.Inc(heights[i])
	}
}
