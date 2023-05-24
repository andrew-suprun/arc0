package ui

import "arch/device"

type Widget interface {
	Constraint() device.Constraint
	Render(device.Device, device.Position, device.Size)
}
