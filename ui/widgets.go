package ui

import "arch/device"

type Widget interface {
	Constraint() Constraint
	Render(*Context, Position, Size)
}

type Context struct {
	Device           device.Device
	Style            device.Style
	MouseTargetAreas []MouseTargetArea
	ScrollAreas      []ScrollArea
}

type MouseTargetArea struct {
	Pos  Position
	Size Size
}

type ScrollArea struct {
	Pos  Position
	Size Size
}
