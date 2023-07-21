package widgets

import (
	"fmt"
	"strings"
)

type column struct {
	constraint Constraint
	widgets    []Widget
}

func Column(constraint Constraint, widgets ...Widget) Widget {
	return column{constraint: constraint, widgets: widgets}
}

func (c column) Constraint() Constraint {
	return c.constraint
}

func (c column) Render(screen *Screen, pos Position, size Size) {
	sizes := make([]int, len(c.widgets))
	flexes := make([]int, len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraint().Height
		flexes[i] = widget.Constraint().Y
	}
	heights := calcSizes(size.Height, sizes, flexes)
	for i, widget := range c.widgets {
		widget.Render(screen, Position{X: pos.X, Y: pos.Y}, Size{Width: size.Width, Height: heights[i]})
		pos.Y += heights[i]
	}
}

func (c column) String() string { return toString(c) }

func (c column) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Column(%s\n", c.constraint)
	for _, w := range c.widgets {
		w.ToString(buf, offset+"| ")
	}
}
