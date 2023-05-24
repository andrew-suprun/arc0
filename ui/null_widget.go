package ui

import "arch/device"

type NullWidget struct{}

func (w NullWidget) Constraint() device.Constraint {
	return device.Constraint{Size: device.Size{Width: 0, Height: 0}, Flex: device.Flex{X: 0, Y: 0}}
}

func (w NullWidget) Render(device.Device, device.Position, device.Size) {}
