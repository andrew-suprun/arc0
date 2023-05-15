package ui

import "arch/device"

type styled struct {
	style  device.Style
	widget Widget
}

func Styled(style device.Style, widget Widget) Widget {
	return styled{style: style, widget: widget}
}

func (s styled) Constraint() Constraint {
	return s.widget.Constraint()
}

// TODO: remove style argument
func (s styled) Render(ctx *Context, pos Position, size Size) {
	currentStyle := ctx.Style
	ctx.Style = s.style
	s.widget.Render(ctx, pos, size)
	ctx.Style = currentStyle
}
