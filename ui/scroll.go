package ui

type scroll struct {
	command    any
	constraint Constraint
	widget     func(size Size) Widget
}

func Scroll(command any, constraint Constraint, widget func(size Size) Widget) Widget {
	return scroll{command, constraint, widget}
}

func (s scroll) Constraint() Constraint {
	return s.constraint
}

func (s scroll) Render(ctx *Context, pos Position, size Size) {
	ctx.ScrollAreas = append(ctx.ScrollAreas, ScrollArea{s.command, pos, size})
	widget := s.widget(size)
	widget.Render(ctx, pos, size)
}
