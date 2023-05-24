package ui

import (
	"arch/device"
	"math"
)

type progressBar struct {
	value float64
	width int
	flex  int
}

func ProgressBar(value float64) *progressBar {
	return &progressBar{
		value: value,
		width: 0,
		flex:  1,
	}
}

func (t *progressBar) Width(width int) *progressBar {
	t.width = width
	return t
}

func (t *progressBar) Flex(flex int) *progressBar {
	t.flex = flex
	return t
}

func (pb progressBar) Constraint() device.Constraint {
	return device.Constraint{Size: device.Size{Width: pb.width, Height: 1}, Flex: device.Flex{X: pb.flex, Y: 0}}
}

func (pb progressBar) Render(d device.Device, pos device.Position, size device.Size) {
	if size.Width < 1 {
		return
	}

	runes := make([]rune, size.Width)
	progress := int(math.Round(float64(size.Width*8) * float64(pb.value)))
	idx := 0
	for ; idx < progress/8; idx++ {
		runes[idx] = '█'
	}
	if progress%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < int(size.Width); idx++ {
		runes[idx] = ' '
	}

	d.Text(runes, pos)
}
