package widgets

import (
	"arch/model"
	"fmt"
	"strings"
)

type mouseTarget struct {
	event  model.MouseTarget
	widget Widget
}

func MouseTarget(cmd any, widget Widget) Widget {
	return mouseTarget{event: model.MouseTarget{Command: cmd}, widget: widget}
}

func (t mouseTarget) Constraint() Constraint {
	return t.widget.Constraint()
}

func (t mouseTarget) Render(renderer Renderer, pos Position, size Size) {
	renderer.AddMouseTarget(t.event, pos, size)
	t.widget.Render(renderer, pos, size)
}

func (t mouseTarget) String() string { return toString(t) }

func (t mouseTarget) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"MouseTarget(%s\n", t.event)
	t.widget.ToString(buf, offset+"| ")
}
