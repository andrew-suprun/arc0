package ui

import (
	"arch/device"
	"math"
)

type row struct {
	widgets []Widget
}

func Row(ws ...Widget) Widget {
	return row{ws}
}

func (r row) Constraint() device.Constraint {
	width, flex := 0, 0
	for _, widget := range r.widgets {
		c := widget.Constraint()
		width += c.Size.Width
		flex += c.Flex.X
	}
	return device.Constraint{Size: device.Size{Width: width, Height: 1}, Flex: device.Flex{X: 1, Y: 0}}
}

func (r row) Render(d device.Device, pos device.Position, size device.Size) {
	sizes := make([]int, len(r.widgets))
	flexes := make([]int, len(r.widgets))
	for i, widget := range r.widgets {
		sizes[i] = widget.Constraint().Width
		flexes[i] = widget.Constraint().X
	}
	widths := calcSizes(size.Width, sizes, flexes)
	for i, widget := range r.widgets {
		widget.Render(d, pos, device.Size{Width: widths[i], Height: size.Height})
		pos.X += widths[i]
	}
}

func calcSizes(targetSize int, sizes []int, flexes []int) []int {
	result := make([]int, len(sizes))
	totalSize, totalFlex := 0, 0
	for i, size := range sizes {
		result[i] = size
		totalSize += size
		totalFlex += flexes[i]
	}
	for totalSize > targetSize {
		idx := 0
		maxSize := result[0]
		for i, size := range result {
			if maxSize < size {
				maxSize = size
				idx = i
			}
		}
		result[idx]--
		totalSize--
	}

	if totalFlex == 0 {
		return result
	}

	if totalSize < targetSize {
		diff := targetSize - totalSize
		remainders := make([]float64, len(sizes))
		for i, flex := range flexes {
			rate := float64(diff*flex) / float64(totalFlex)
			remainders[i] = rate - math.Floor(rate)
			result[i] += int(rate)
		}
		totalSize := 0
		for _, size := range result {
			totalSize += size
		}
		for i := range result {
			if totalSize == targetSize {
				break
			}
			if flexes[i] > 0 {
				result[i]++
				totalSize++
			}
		}
		for i := range result {
			if totalSize == targetSize {
				break
			}
			if flexes[i] == 0 {
				result[i]++
				totalSize++
			}
		}
	}

	return result
}
