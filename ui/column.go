package ui

import "arch/device"

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
	return column{height: height, flex: flex, widgets: widgets}
}

func (c column) Constraint() device.Constraint {
	return device.Constraint{Size: device.Size{Width: 0, Height: c.height}, Flex: device.Flex{X: 1, Y: c.flex}}
}

func (c column) Render(d device.Device, pos device.Position, size device.Size) {
	sizes := make([]int, len(c.widgets))
	flexes := make([]int, len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraint().Height
		flexes[i] = widget.Constraint().Y
	}
	heights := calcSizes(size.Height, sizes, flexes)
	y := 0
	for i, widget := range c.widgets {
		widget.Render(d, device.Position{X: pos.X, Y: pos.Y + y}, device.Size{Width: size.Width, Height: heights[i]})
		y += heights[i]
	}
}
