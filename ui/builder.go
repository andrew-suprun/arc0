package ui

import (
	"math"
)

type Builder struct {
	width, height int
	x, y          int
	style         Style
	mouseTarget   any
	fields        []Drawable
	screen        Screen
	sizes         []int
}

type Field struct {
	Width, Flex int
}

type Drawable interface {
	Draw(width int) []rune
	Style() Style
	MouseTarget() any
}

func NewBuilder(width, height int) *Builder {
	return &Builder{width: width, height: height}
}

func (b *Builder) AddText(txt string) {
	b.fields = append(b.fields, Text{text: txt, style: b.style, mouseTarget: b.mouseTarget})
}

func (b *Builder) AddRText(txt string) {
	b.fields = append(b.fields, RText{text: txt, style: b.style, mouseTarget: b.mouseTarget})
}

func (b *Builder) AddProgressBar(value float64) {
	b.fields = append(b.fields, ProgressBar{value: value, style: b.style})
}

func (b *Builder) SetStyle(style Style) {
	b.style = style
}

func (b *Builder) SetMouseTarget(cmd any) {
	b.mouseTarget = cmd
}

func (b *Builder) SetLayout(fields ...Field) {
	b.sizes = make([]int, len(fields))
	totalWidth, totalFlex := 0, 0
	for i, field := range fields {
		b.sizes[i] = field.Width
		totalWidth += field.Width
		totalFlex += field.Flex
	}
	diff := b.width - totalWidth
	if diff <= 0 {
		return
	}

	totalInc := 0
	for i, field := range fields {
		inc := diff * field.Flex / totalFlex
		b.sizes[i] += inc
		totalInc += inc
	}
	diff -= totalInc
	for i := range fields {
		if fields[i].Flex == 0 {
			continue
		}
		if diff == 0 {
			return
		}
		b.sizes[i] += 1
		diff--
	}
}

func (b *Builder) LayoutLine() {
	for i, field := range b.fields {
		segment := Segment{
			X:           b.x,
			Y:           b.y,
			Runes:       field.Draw(b.sizes[i]),
			Style:       field.Style(),
			MouseTarget: field.MouseTarget(),
		}

		b.screen = append(b.screen, segment)
		b.x += len(segment.Runes)
	}
	b.fields = b.fields[:0]
	b.NewLine()
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

type Text struct {
	text        string
	style       Style
	mouseTarget any
}

func (t Text) Draw(width int) []rune {
	if width < 1 {
		return nil
	}
	runes := []rune(t.text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
		return runes
	}

	diff := width - len(runes)
	idx := 0
	result := make([]rune, width)

	for i := range runes {
		result[idx] = runes[i]
		idx++
	}

	if diff > 0 {
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

func (t Text) Style() Style {
	return t.style
}

func (t Text) MouseTarget() any {
	return t.mouseTarget
}

type RText struct {
	text        string
	style       Style
	mouseTarget any
}

func (t RText) Draw(width int) []rune {
	if width < 1 {
		return nil
	}
	runes := []rune(t.text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
		return runes
	}

	diff := width - len(runes)
	idx := 0
	result := make([]rune, width)
	if diff > 0 {
		for i := 0; i < diff; i++ {
			result[idx] = ' '
			idx++
		}
	}

	for i := range runes {
		result[idx] = runes[i]
		idx++
	}

	for ; idx < width; idx++ {
		result[idx] = ' '
	}

	return result
}

func (t RText) Style() Style {
	return t.style
}

func (t RText) MouseTarget() any {
	return t.mouseTarget
}

type ProgressBar struct {
	value float64
	style Style
}

func (pb ProgressBar) Draw(width int) []rune {
	result := make([]rune, width)
	progress := int(math.Round(float64(width*8) * pb.value))
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

func (pb ProgressBar) Style() Style {
	return pb.style
}

func (pb ProgressBar) MouseTarget() any {
	return nil
}
