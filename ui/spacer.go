package ui

import "arch/device"

type Spacer struct{}

func (w Spacer) Constraint() device.Constraint {
	return device.Constraint{Size: device.Size{Width: 0, Height: 0}, Flex: device.Flex{X: 1, Y: 1}}
}

func (w Spacer) Render(d device.Device, pos device.Position, size device.Size) {
	runes := make([]rune, size.Width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(size.Height); i++ {
		d.Text(runes, device.Position{X: pos.X, Y: pos.Y + i})
	}
}
