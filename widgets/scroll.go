package widgets

import (
	"arch/events"
	"fmt"
	"strings"
)

type scroll struct {
	event      events.Scroll
	constraint Constraint
	widget     func(size Size) Widget
}

// TODO: Separate Scroll into Scroll and Sized
func Scroll(event events.Scroll, constraint Constraint, widget func(size Size) Widget) Widget {
	return scroll{event: event, constraint: constraint, widget: widget}
}

func (s scroll) Constraint() Constraint {
	return s.constraint
}

func (s scroll) Render(renderer Renderer, pos Position, size Size) {
	renderer.AddScrollArea(s.event, pos, size)
	widget := s.widget(size)
	widget.Render(renderer, pos, size)
}

func (s scroll) String() string { return toString(s) }

func (s scroll) ToString(buf *strings.Builder, offset string) {
	fmt.Fprintf(buf, "%sScroll(%s\n", offset, s.event)
	fmt.Fprintf(buf, "%s| %s\n", offset, s.constraint)
	widget := s.widget(Size{80, 3})
	widget.ToString(buf, offset+"| ")
}
