package widgets

import (
	"fmt"
	"strings"
)

type mouseTarget struct {
	command any
	widget  Widget
}

func MouseTarget(cmd any, widget Widget) Widget {
	return mouseTarget{command: cmd, widget: widget}
}

func (t mouseTarget) Constraint() Constraint {
	return t.widget.Constraint()
}

func (t mouseTarget) Render(screen *Screen, pos Position, size Size) {
	screen.MouseTargets = append(screen.MouseTargets, MouseTargetArea{Command: t.command, Position: pos, Size: size})
	t.widget.Render(screen, pos, size)
}

func (t mouseTarget) String() string { return toString(t) }

func (t mouseTarget) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"MouseTarget(%v\n", t.command)
	t.widget.ToString(buf, offset+"| ")
	fmt.Fprintf(buf, offset+")\n", t.command)
}
