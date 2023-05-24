package ui

import "arch/device"

type styled struct {
	style  device.Style
	widget Widget
}

func Styled(style device.Style, widget Widget) Widget {
	return styled{style: style, widget: widget}
}

func (s styled) Constraint() device.Constraint {
	return s.widget.Constraint()
}

func (s styled) Render(d device.Device, pos device.Position, size device.Size) {
	currentStyle := d.CurrentStyle()
	d.SetStyle(s.style)
	s.widget.Render(d, pos, size)
	d.SetStyle(currentStyle)
}
