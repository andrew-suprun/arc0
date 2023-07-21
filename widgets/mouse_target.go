package widgets

import (
	m "arch/model"
	"fmt"
	"strings"
)

type mouseTarget struct {
	event  m.MouseTarget
	widget Widget
}

func MouseTarget(cmd any, widget Widget) Widget {
	return mouseTarget{event: m.MouseTarget{Command: cmd}, widget: widget}
}

func (t mouseTarget) Constraint() Constraint {
	return t.widget.Constraint()
}

func (t mouseTarget) Render(screen *Screen, pos Position, size Size) {
	screen.MouseTargets = append(screen.MouseTargets, MouseTargetArea{Command: t.event, Position: pos, Size: size})
	t.widget.Render(screen, pos, size)
}

func (t mouseTarget) String() string { return toString(t) }

func (t mouseTarget) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"MouseTarget(%s\n", t.event)
	t.widget.ToString(buf, offset+"| ")
}
