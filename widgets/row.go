package widgets

import (
	"fmt"
	"strings"
)

type row struct {
	constraint Constraint
	widgets    []Widget
}

func Row(constraint Constraint, ws ...Widget) Widget {
	return row{constraint: constraint, widgets: ws}
}

func (r row) Constraint() Constraint {
	return r.constraint
}

func (r row) Render(screen *Screen, pos Position, size Size) {
	sizes := make([]int, len(r.widgets))
	flexes := make([]int, len(r.widgets))
	for i, widget := range r.widgets {
		sizes[i] = widget.Constraint().Width
		flexes[i] = widget.Constraint().X
	}
	widths := calcSizes(size.Width, sizes, flexes)
	for i, widget := range r.widgets {
		widget.Render(screen, pos, Size{Width: widths[i], Height: size.Height})
		pos.X += widths[i]
	}
}

func (r row) String() string { return toString(r) }

func (r row) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Row(%s\n", r.constraint)
	for _, w := range r.widgets {
		w.ToString(buf, offset+"| ")
	}
}
