package app

import (
	"arch/ui"
	"math"
)

type builder struct {
	width, height int
	x, y          int
	defaultStyle  ui.Style
	screen        ui.Screen
	fields        []field
}

type field struct {
	size  int
	style ui.Style
	align alignment
	flex  bool
}

type alignment byte

const (
	left alignment = iota
	right
)

type drawable interface {
	draw(width int, alight alignment) []rune
}

func newBuilder(width, height int) *builder {
	return &builder{width: width, height: height}
}

func (b *builder) setDefaultStyle(style ui.Style) {
	b.defaultStyle = style
}

func (b *builder) setLayout(fields ...field) {
	b.fields = fields
	if len(fields) == 0 || len(fields) > b.width {
		return
	}

	layoutWidth := 0
	for i := range fields {
		layoutWidth += fields[i].size
	}
	for layoutWidth < b.width {
		shortestFixedField, shortestFixedFieldIdx := math.MaxInt, -1
		shortestFlexField, shortestFlexFieldIdx := math.MaxInt, -1
		for j := range fields {
			if fields[j].flex {
				if shortestFlexField > fields[j].size {
					shortestFlexField = fields[j].size
					shortestFlexFieldIdx = j
				}
			} else {
				if shortestFixedField > fields[j].size {
					shortestFixedField = fields[j].size
					shortestFixedFieldIdx = j
				}
			}
		}
		if shortestFlexFieldIdx != -1 {
			fields[shortestFlexFieldIdx].size++
		} else {
			fields[shortestFixedFieldIdx].size++
		}
		layoutWidth++
	}
	for layoutWidth > b.width {
		longestFixedField, longestFixedFieldIdx := 0, -1
		longestFlexField, longestFlexFieldIdx := 0, -1
		for j := range fields {
			if fields[j].flex {
				if longestFlexField < fields[j].size {
					longestFlexField = fields[j].size
					longestFlexFieldIdx = j
				}
			} else {
				if longestFixedField < fields[j].size {
					longestFixedField = fields[j].size
					longestFixedFieldIdx = j
				}
			}
		}

		if longestFlexFieldIdx != -1 && fields[longestFlexFieldIdx].size > 1 {
			fields[longestFlexFieldIdx].size--
		} else if longestFixedFieldIdx == -1 {
			return
		} else {
			if fields[longestFixedFieldIdx].size > 1 {
				fields[longestFixedFieldIdx].size--
			}
		}
		layoutWidth--
	}
}

func (b *builder) drawTexts(texts ...string) {
	drawables := make([]drawable, len(texts))
	for i := range texts {
		drawables[i] = text(texts[i])
	}
	b.drawLine(drawables...)
}

func (b *builder) drawLine(texts ...drawable) {
	for i := range texts {
		segment := ui.Segment{
			X:     b.x,
			Y:     b.y,
			Runes: texts[i].draw(b.fields[i].size, b.fields[i].align),
			Style: b.style(b.fields[i].style),
		}

		b.screen = append(b.screen, segment)
		b.x += len(segment.Runes)
	}
	b.newLine()
}

func (b *builder) style(style ui.Style) ui.Style {
	if style == ui.NoStyle {
		return b.defaultStyle
	}
	return style
}

func (b *builder) newLine() {
	b.x = 0
	b.y++
}

func (b *builder) setLine(y int) {
	b.x = 0
	b.y = y
}

func (b *builder) getScreen() ui.Screen {
	return b.screen
}

type text string

func (t text) draw(width int, align alignment) []rune {
	if width < 1 {
		return nil
	}
	runes := []rune(t)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
		return runes
	}

	diff := width - len(runes)
	idx := 0
	result := make([]rune, width)
	if diff > 0 && align == right {
		for i := 0; i < diff; i++ {
			result[idx] = ' '
			idx++
		}
	}

	for i := range runes {
		result[idx] = runes[i]
		idx++
	}

	if diff > 0 && align == left {
		for i := 0; i < diff; i++ {
			result[idx] = ' '
			idx++
		}
	}
	for ; idx < width; idx++ {
		result[idx] = ' '
	}

	return result
}

type progressBar float64

func (pb progressBar) draw(width int, _ alignment) []rune {
	result := make([]rune, width)
	progress := int(math.Round(float64(width*8) * float64(pb)))
	idx := 0
	for ; idx < progress/8; idx++ {
		result[idx] = '█'
	}
	if progress%8 > 0 {
		result[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < width; idx++ {
		result[idx] = ' '
	}
	return result
}
