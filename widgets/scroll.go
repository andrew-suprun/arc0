package widgets

import (
	m "arch/model"
	"fmt"
	"strings"
)

type scroll struct {
	event      m.Scroll
	constraint Constraint
	widget     func(size Size) Widget
}

// TODO: Separate Scroll into Scroll and Sized
func Scroll(event m.Scroll, constraint Constraint, widget func(size Size) Widget) Widget {
	return scroll{event: event, constraint: constraint, widget: widget}
}

func (s scroll) Constraint() Constraint {
	return s.constraint
}

func (s scroll) Render(screen *Screen, pos Position, size Size) {
	screen.ScrollAreas = append(screen.ScrollAreas, ScrollArea{Command: s.event, Position: pos, Size: size})
	widget := s.widget(size)
	widget.Render(screen, pos, size)
}

func (s scroll) String() string { return toString(s) }

func (s scroll) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, "%sScroll(%s\n", offset, s.event)
	fmt.Fprintf(buf, "%s| %s\n", offset, s.constraint)
	widget := s.widget(Size{80, 3})
	widget.ToString(buf, offset+"| ")
}
