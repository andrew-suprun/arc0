package ui

import "math"

type progressBar struct {
	value float64
	width X
	flex  Flex
}

func ProgressBar(value float64, width X, flex Flex) progressBar {
	return progressBar{
		value: value,
		width: width,
		flex:  flex,
	}
}

func (pb progressBar) Constraints() Constraints {
	return MakeConstraints(pb.width, pb.flex, 1, 0)
}

func (pb progressBar) Flex() int {
	return 2
}

func (pb progressBar) Render(renderer Renderer, x X, y Y, width X, _ Y, style Style) {
	if width < 1 {
		return
	}

	runes := make([]rune, width)
	progress := int(math.Round(float64(width*8) * float64(pb.value)))
	idx := 0
	for ; idx < progress/8; idx++ {
		runes[idx] = '█'
	}
	if progress%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < int(width); idx++ {
		runes[idx] = ' '
	}

	renderer.Text(runes, x, y, style)
}
