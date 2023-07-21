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

func (s styled) Render(screen *Screen, pos Position, size Size) {
	currentStyle := screen.Style
	screen.Style = s.style
	s.widget.Render(screen, pos, size)
	screen.Style = currentStyle
}

func (s styled) String() string { return toString(s) }

func (s styled) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, offset+"Styled(%s\n", s.style)
	s.widget.ToString(buf, offset+"| ")
}
