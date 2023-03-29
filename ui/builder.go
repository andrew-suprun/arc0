package ui

import (
	"math"
)

type Builder struct {
	width, height int
	x, y          int
	defaultStyle  Style
	screen        Screen
	fields        []Field
}

type Field struct {
	Size  int
	Style Style
	Align Alignment
	Flex  bool
}

type Alignment byte

const (
	Left Alignment = iota
	Right
)

type Drawable interface {
	Draw(width int, alight Alignment) []rune
}

func NewBuilder(width, height int) *Builder {
	return &Builder{width: width, height: height}
}

func (b *Builder) SetDefaultStyle(style Style) {
	b.defaultStyle = style
}

func (b *Builder) SetLayout(fields ...Field) {
	b.fields = fields
	if len(fields) == 0 || len(fields) > b.width {
		return
	}

	layoutWidth := 0
	for i := range fields {
		layoutWidth += fields[i].Size
	}
	for layoutWidth < b.width {
		shortestFixedField, shortestFixedFieldIdx := math.MaxInt, -1
		shortestFlexField, shortestFlexFieldIdx := math.MaxInt, -1
		for j := range fields {
			if fields[j].Flex {
				if shortestFlexField > fields[j].Size {
					shortestFlexField = fields[j].Size
					shortestFlexFieldIdx = j
				}
			} else {
				if shortestFixedField > fields[j].Size {
					shortestFixedField = fields[j].Size
					shortestFixedFieldIdx = j
				}
			}
		}
		if shortestFlexFieldIdx != -1 {
			fields[shortestFlexFieldIdx].Size++
		} else {
			fields[shortestFixedFieldIdx].Size++
		}
		layoutWidth++
	}
	for layoutWidth > b.width {
		longestFixedField, longestFixedFieldIdx := 0, -1
		longestFlexField, longestFlexFieldIdx := 0, -1
		for j := range fields {
			if fields[j].Flex {
				if longestFlexField < fields[j].Size {
					longestFlexField = fields[j].Size
					longestFlexFieldIdx = j
				}
			} else {
				if longestFixedField < fields[j].Size {
					longestFixedField = fields[j].Size
					longestFixedFieldIdx = j
				}
			}
		}

		if longestFlexFieldIdx != -1 && fields[longestFlexFieldIdx].Size > 1 {
			fields[longestFlexFieldIdx].Size--
		} else if longestFixedFieldIdx == -1 {
			return
		} else {
			if fields[longestFixedFieldIdx].Size > 1 {
				fields[longestFixedFieldIdx].Size--
			}
		}
		layoutWidth--
	}
}

func (b *Builder) DrawTexts(texts ...string) {
	drawables := make([]Drawable, len(texts))
	for i := range texts {
		drawables[i] = Text(texts[i])
	}
	b.DrawLine(drawables...)
}

func (b *Builder) DrawLine(texts ...Drawable) {
	for i := range texts {
		segment := Segment{
			X:     b.x,
			Y:     b.y,
			Runes: texts[i].Draw(b.fields[i].Size, b.fields[i].Align),
			Style: b.style(b.fields[i].Style),
		}

		b.screen = append(b.screen, segment)
		b.x += len(segment.Runes)
	}
	b.NewLine()
}

func (b *Builder) style(style Style) Style {
	if style == NoStyle {
		return b.defaultStyle
	}
	return style
}

func (b *Builder) NewLine() {
	b.x = 0
	b.y++
}

func (b *Builder) SetLine(y int) {
	b.x = 0
	b.y = y
}

func (b *Builder) GetScreen() Screen {
	return b.screen
}

type Text string

func (t Text) Draw(width int, align Alignment) []rune {
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
	if diff > 0 && align == Right {
		for i := 0; i < diff; i++ {
			result[idx] = ' '
			idx++
		}
	}

	for i := range runes {
		result[idx] = runes[i]
		idx++
	}

	if diff > 0 && align == Left {
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

type ProgressBar float64

func (pb ProgressBar) Draw(width int, _ Alignment) []rune {
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
