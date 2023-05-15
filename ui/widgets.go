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
	Command any
	Pos     Position
	Size    Size
}

type ScrollArea struct {
	Command any
	Pos     Position
	Size    Size
}

func (ctx *Context) Reset() {
	ctx.MouseTargetAreas = ctx.MouseTargetAreas[:0]
	ctx.ScrollAreas = ctx.ScrollAreas[:0]
}
