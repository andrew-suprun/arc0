package ui

type column struct {
	constraint Constraint[Y]
	widgets    []Widget
}

func Column(flex Flex, widgets ...Widget) Widget {
	height := Y(0)
	for _, widget := range widgets {
		height += widget.Constraints().Height.Size
	}
	return column{Constraint[Y]{height, flex}, widgets}
}

func (c column) Constraints() Constraints {
	return Constraints{Width: Constraint[X]{0, 1}, Height: c.constraint}
}

func (c column) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
	sizes := make([]Constraint[Y], len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraints().Height
	}
	heights := calcSizes(height, sizes)
	for i, widget := range c.widgets {
		widget.Render(renderer, x, y, width, heights[i], style)
		y += heights[i]
	}
}
