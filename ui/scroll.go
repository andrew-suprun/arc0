package ui

import "arch/device"

type scroll struct {
	command    any
	constraint device.Constraint
	widget     func(size device.Size) Widget
}

func Scroll(command any, constraint device.Constraint, widget func(size device.Size) Widget) Widget {
	return scroll{command: command, constraint: constraint, widget: widget}
}

func (s scroll) Constraint() device.Constraint {
	return s.constraint
}

func (s scroll) Render(d device.Device, pos device.Position, size device.Size) {
	d.AddScrollArea(s.command, pos, size)
	widget := s.widget(size)
	widget.Render(d, pos, size)
}
