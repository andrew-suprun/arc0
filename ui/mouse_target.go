package ui

import "arch/device"

type mouse_target struct {
	command any
	widget  Widget
}

func MouseTarget(command any, widget Widget) Widget {
	return mouse_target{command: command, widget: widget}
}

func (s mouse_target) Constraint() device.Constraint {
	return s.widget.Constraint()
}

func (t mouse_target) Render(d device.Device, pos device.Position, size device.Size) {
	d.AddMouseTarget(t.command, pos, size)
	t.widget.Render(d, pos, size)
}
