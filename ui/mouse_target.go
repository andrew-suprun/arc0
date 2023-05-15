package ui

type mouse_target struct {
	command any
	widget  Widget
}

func MouseTarget(command any, widget Widget) Widget {
	return mouse_target{command: command, widget: widget}
}

func (s mouse_target) Constraint() Constraint {
	return s.widget.Constraint()
}

func (t mouse_target) Render(ctx *Context, pos Position, size Size) {
	ctx.MouseTargetAreas = append(ctx.MouseTargetAreas, MouseTargetArea{t.command, pos, size})
	t.widget.Render(ctx, pos, size)
}
