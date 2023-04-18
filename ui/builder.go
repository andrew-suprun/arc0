package ui

import (
	"log"
	"math"
)

type view struct {
	width, height int
	col, line     int
	x, y          int
	layoutIdx     int
	style         Style
	mouseTarget   any
	layout        []Field
	segments      []Segment
}

func View(leftCol, topLine, width, height int, contents ...Widget) []Segment {
	view := &view{
		col:    leftCol,
		line:   topLine,
		width:  width,
		height: height,
	}
	for _, content := range contents {
		view.draw(content)
	}
	return view.segments
}

type Widget interface {
	widget()
}

func (v *view) draw(content Widget) {
	switch content := content.(type) {
	case layout:
		v.drawLayout(content)

	case drawable:
		v.drawDrawable(content)

	case line:
		v.drawLine(content)

	case style:
		v.drawStyle(content)

	case mouseTarget:
		v.drawMouseTarget(content)

	default:
		log.Printf("UNHANDLED: [%T] %#v\n", content, content)
		panic("UNHANDLED")
	}
}

type layout struct {
	layout   []Field
	contents []Widget
}

func (l layout) widget() {}

type Field struct {
	Width int
	Flex  int
	Text  string
}

func (v *view) drawLayout(l layout) {
	currentLayout := v.layout
	defer func() {
		for _, child := range l.contents {
			v.draw(child)
		}
		v.layout = currentLayout
	}()

	v.layout = l.layout
	totalWidth, totalFlex := 0, 0
	for i, field := range v.layout {
		if field.Text != "" {
			runes := []rune(field.Text)
			v.layout[i].Width = len(runes)
			field.Flex = 0
		}
		totalWidth += v.layout[i].Width
		totalFlex += field.Flex
	}
	diff := v.width - totalWidth
	if diff <= 0 {
		return
	}

	totalInc := 0
	for i, field := range v.layout {
		inc := diff * field.Flex / totalFlex
		v.layout[i].Width += inc
		totalInc += inc
	}
	diff -= totalInc
	for i := range l.layout {
		if l.layout[i].Flex == 0 {
			continue
		}
		if diff == 0 {
			log.Printf("layout: %#v", v.layout)
			return
		}
		v.layout[i].Width += 1
		diff--
	}
}

type drawable interface {
	Widget
	runes(width int) []rune
}

func (v *view) drawDrawable(content drawable) {
	if len(v.layout) <= v.layoutIdx {
		return
	}

	runes := content.runes(v.layout[v.layoutIdx].Width)
	v.segments = append(v.segments, Segment{
		X:           v.x + v.col,
		Y:           v.y + v.line,
		Runes:       runes,
		Style:       v.style,
		MouseTarget: v.mouseTarget,
	})
	v.x += len(runes)
}

type line []Widget

func (l line) widget() {}

func Line(contents ...Widget) line {
	return contents
}

func (v *view) drawLine(content line) {
	if v.y >= v.height {
		return
	}

	i := 0
	field := Field{}
	for v.layoutIdx, field = range v.layout {
		if field.Text != "" {
			v.draw(Text(field.Text))
		} else {
			v.draw(content[i])
			i++
		}
	}

	v.x = 0
	v.y++
}

type style struct {
	style    Style
	contents []Widget
}

func (s style) widget() {}

func Styled(styl Style, contents ...Widget) style {
	return style{
		style:    styl,
		contents: contents,
	}
}

func (v *view) drawStyle(content style) {
	currentStyle := v.style
	v.style = content.style
	for _, child := range content.contents {
		v.draw(child)
	}
	v.style = currentStyle
}

type mouseTarget struct {
	command  any
	contents []Widget
}

func (t mouseTarget) widget() {}

func MouseTarget(command any, contents ...Widget) mouseTarget {
	return mouseTarget{
		command:  command,
		contents: contents,
	}
}

func (v *view) drawMouseTarget(content mouseTarget) {
	currentCommand := v.mouseTarget
	v.mouseTarget = content.command
	for _, child := range content.contents {
		v.draw(child)
	}
	v.mouseTarget = currentCommand
}

func Fixed(width int) Field {
	return Field{Width: width}
}

func Flex(flex int) Field {
	return Field{Flex: flex, Width: 4}
}

func Spacer(flex int) Field {
	return Field{Flex: flex, Width: 0}
}

func Pad(text string) Field {
	return Field{Text: text}
}

func Layout(fields []Field, contents ...Widget) layout {
	return layout{fields, contents}
}

type text string

func (t text) widget() {}

func Text(txt string) text {
	return text(txt)
}

func (t text) runes(width int) []rune {
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

type rText string

func (t rText) widget() {}

func RText(txt string) rText {
	return rText(txt)
}

func (t rText) runes(width int) []rune {
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

type progressBar float64

func (pb progressBar) widget() {}

func ProgressBar(value float64) progressBar {
	return progressBar(value)
}

func (pb progressBar) runes(width int) []rune {
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
