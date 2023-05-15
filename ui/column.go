package ui

type column struct {
	height  int
	flex    int
	widgets []Widget
}

func Column(flex int, widgets ...Widget) Widget {
	height := 0
	for _, widget := range widgets {
		height += widget.Constraint().Size.Height
	}
	return column{height, flex, widgets}
}

func (c column) Constraint() Constraint {
	return Constraint{Size{0, c.height}, Flex{1, c.flex}}
}

func (c column) Render(ctx *Context, pos Position, size Size) {
	sizes := make([]int, len(c.widgets))
	flexes := make([]int, len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraint().Height
		flexes[i] = widget.Constraint().Y
	}
	heights := calcSizes(size.Height, sizes, flexes)
	y := 0
	for i, widget := range c.widgets {
		widget.Render(ctx, Position{pos.X, pos.Y + y}, Size{size.Width, heights[i]})
		y += heights[i]
	}
}
