package widgets

import (
	"fmt"
	"strings"
)

type styled struct {
	style  Style
	widget Widget
}

func Styled(style Style, widget Widget) Widget {
	return styled{style: style, widget: widget}
}

func (s styled) Constraint() Constraint {
	return s.widget.Constraint()
}

func (s styled) Render(renderer Renderer, pos Position, size Size) {
	currentStyle := renderer.CurrentStyle()
	renderer.SetStyle(s.style)
	s.widget.Render(renderer, pos, size)
	renderer.SetStyle(currentStyle)
}

func (s styled) String() string { return toString(s) }

func (s styled) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Styled(%s\n", s.style)
	s.widget.ToString(buf, offset+"| ")
}
