package ui

import (
	"math"
)

type Widget interface {
	Size() Size
	Flex() int
	Render(position Position, size Size, attributes *Attributes) Segments
}

type Size struct {
	Width, Height int
}

// *** Text ***
type text struct {
	runes []rune
	flex  int
}

func Text(txt string, flex int) text {
	return text{[]rune(txt), flex}
}

func (t text) Size() Size {
	return Size{Width: len(t.runes), Height: 1}
}

func (t text) Flex() int {
	return t.flex
}

func (t text) Render(position Position, size Size, attributes *Attributes) Segments {
	if size.Width < 1 {
		return nil
	}
	if len(t.runes) > size.Width {
		t.runes = append(t.runes[:size.Width-1], '…')
	}
	diff := size.Width - len(t.runes)
	for diff > 0 {
		t.runes = append(t.runes, ' ')
		diff--
	}
	return Segments{{
		Position:   position,
		Runes:      t.runes,
		Attributes: attributes,
	}}
}

type progressBar float64

func ProgressBar(value float64) progressBar {
	return progressBar(value)
}

func (pb progressBar) Size() Size {
	return Size{Width: math.MaxInt, Height: 1}
}

func (pb progressBar) Flex() int {
	return 2
}

func (pb progressBar) Render(position Position, size Size, attributes *Attributes) Segments {
	if size.Width < 1 {
		return nil
	}

	runes := make([]rune, size.Width)
	progress := int(math.Round(float64(size.Width*8) * float64(pb)))
	idx := 0
	for ; idx < progress/8; idx++ {
		runes[idx] = '█'
	}
	if progress%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < size.Width; idx++ {
		runes[idx] = ' '
	}

	return Segments{{
		Position:   position,
		Runes:      runes,
		Attributes: attributes,
	}}
}

// *** Styled ***
type styled struct {
	style  Style
	widget Widget
}

func Styled(style Style, widget Widget) Widget {
	return styled{style: style, widget: widget}
}

func (s styled) Size() Size {
	return s.widget.Size()
}

func (s styled) Flex() int {
	return s.widget.Flex()
}

func (s styled) Render(position Position, size Size, attributes *Attributes) Segments {
	return s.widget.Render(position, size, attributes.Style(s.style))
}

// *** row ***
type row struct {
	widgets []Widget
}

func Row(ws ...Widget) Widget {
	return row{ws}
}

func (r row) Size() Size {
	width := 0
	for _, w := range r.widgets {
		width += w.Size().Width
	}
	return Size{Width: width, Height: 1}
}

func (r row) Flex() int {
	flex := 0
	for _, w := range r.widgets {
		if flex < w.Flex() {
			flex = w.Flex()
		}
	}
	return flex
}

func (r row) Render(position Position, size Size, attributes *Attributes) Segments {
	result := Segments{}
	widths := calcWidths(size.Width, r.widgets)
	for i, w := range r.widgets {
		result = append(result, w.Render(position, Size{Width: widths[i], Height: 1}, attributes)...)
		position.X += widths[i]
	}

	return result
}

func calcWidths(width int, widgets []Widget) []int {
	widths := make([]int, len(widgets))

	totalWidth, totalFlex := 0, 0
	fixedWidth, fixedFields := 0, 0
	for i, w := range widgets {
		if w.Flex() == 0 {
			fixedWidth += w.Size().Width
			fixedFields++
		}
		totalFlex += w.Flex()
		widths[i] = w.Size().Width
		totalWidth += w.Size().Width
	}
	if fixedWidth >= width {
		rate := float64(width) / float64(fixedWidth)
		for i := range widgets {
			if widgets[i].Flex() == 0 {
				newWidth := int(float64(widths[i]) * rate)
				fixedWidth += newWidth - widths[i]
				widths[i] = newWidth
			} else {
				widths[i] = 0
			}
		}
		for i, w := range widgets {
			if fixedWidth == width {
				break
			}
			if w.Flex() == 0 {
				widths[i]++
				fixedWidth++
			}
		}
	} else {
		rate := float64(width-fixedWidth) / float64(totalFlex)
		for i, w := range widgets {
			if w.Flex() > 0 {
				newWidth := int(rate * float64(w.Flex()))
				totalWidth += newWidth - widths[i] // ???
				widths[i] = newWidth
			}
		}
		for i, w := range widgets {
			if totalWidth == width {
				break
			}
			if w.Flex() > 0 {
				widths[i]++
				totalWidth++
			}
		}
	}

	newWidth := 0
	for i := range widths {
		newWidth += widths[i]
	}

	return widths
}

// *** Column ***
type Column struct {
	widgets []Widget
}

func (c Column) Size() Size {
	width, height := 0, 0
	for _, w := range c.widgets {
		if width < w.Size().Width {
			width = w.Size().Width
		}
		height += w.Size().Height
	}
	return Size{Width: width, Height: height}
}

func (c Column) Flex() int {
	flex := 0
	for _, w := range c.widgets {
		if flex < w.Flex() {
			flex = w.Flex()
		}
	}
	return flex
}

func (c Column) Render(position Position, size Size, attributes *Attributes) Segments {
	// TODO
	return nil
}

// ################

// type view struct {
// 	width, height int
// 	col, line     int
// 	x, y          int
// 	layoutIdx     int
// 	style         Style
// 	mouseTarget   any
// 	layout        Fields
// 	segments      Segments
// }

// func View(leftCol, topLine, width, height int, contents ...Widget) Segments {
// 	view := &view{
// 		col:    leftCol,
// 		line:   topLine,
// 		width:  width,
// 		height: height,
// 	}
// 	for _, content := range contents {
// 		view.draw(content)
// 	}
// 	return view.segments
// }

// func (v *view) draw(content Widget) {
// 	switch content := content.(type) {
// 	case layout:
// 		v.drawLayout(content)

// 	case drawable:
// 		v.drawDrawable(content)

// 	case line:
// 		v.drawLine(content)

// 	case style:
// 		v.drawStyle(content)

// 	case mouseTarget:
// 		v.drawMouseTarget(content)

// 	default:
// 		log.Printf("UNHANDLED: [%T] %#v\n", content, content)
// 		panic("UNHANDLED")
// 	}
// }

// type layout struct {
// 	layout   Fields
// 	contents []Widget
// }

// type Field struct {
// 	Width int
// 	Flex  int
// 	Text  string
// }

// type Fields []Field

// func (v *view) drawLayout(l layout) {
// 	currentLayout := v.layout
// 	defer func() {
// 		for _, child := range l.contents {
// 			v.draw(child)
// 		}
// 		v.layout = currentLayout
// 	}()

// 	v.layout = l.layout
// 	totalWidth, totalFlex := 0, 0
// 	for i, field := range v.layout {
// 		if field.Text != "" {
// 			runes := []rune(field.Text)
// 			v.layout[i].Width = len(runes)
// 			field.Flex = 0
// 		}
// 		totalWidth += v.layout[i].Width
// 		totalFlex += field.Flex
// 	}
// 	diff := v.width - totalWidth
// 	if diff <= 0 {
// 		return
// 	}

// 	totalInc := 0
// 	for i, field := range v.layout {
// 		inc := diff * field.Flex / totalFlex
// 		v.layout[i].Width += inc
// 		totalInc += inc
// 	}
// 	diff -= totalInc
// 	for i := range l.layout {
// 		if l.layout[i].Flex == 0 {
// 			continue
// 		}
// 		if diff == 0 {
// 			return
// 		}
// 		v.layout[i].Width += 1
// 		diff--
// 	}
// }

// type drawable interface {
// 	Widget
// 	runes(width int) []rune
// }

// func (v *view) drawDrawable(content drawable) {
// 	if len(v.layout) <= v.layoutIdx {
// 		return
// 	}

// 	runes := content.runes(v.layout[v.layoutIdx].Width)
// 	v.segments = append(v.segments, Segment{
// 		X:           v.x + v.col,
// 		Y:           v.y + v.line,
// 		Runes:       runes,
// 		Style:       v.style,
// 		MouseTarget: v.mouseTarget,
// 	})
// 	v.x += len(runes)
// }

// type line []Widget

// func Line(contents ...Widget) line {
// 	return contents
// }

// func (v *view) drawLine(content line) {
// 	if v.y >= v.height {
// 		return
// 	}

// 	i := 0
// 	field := Field{}
// 	for v.layoutIdx, field = range v.layout {
// 		if field.Text != "" {
// 			v.draw(Text(field.Text))
// 		} else {
// 			v.draw(content[i])
// 			i++
// 		}
// 	}

// 	v.x = 0
// 	v.y++
// }

// type style struct {
// 	style    Style
// 	contents []Widget
// }

// func Styled(styl Style, contents ...Widget) style {
// 	return style{
// 		style:    styl,
// 		contents: contents,
// 	}
// }

// func (v *view) drawStyle(content style) {
// 	currentStyle := v.style
// 	v.style = content.style
// 	for _, child := range content.contents {
// 		v.draw(child)
// 	}
// 	v.style = currentStyle
// }

// type mouseTarget struct {
// 	command  any
// 	contents []Widget
// }

// func MouseTarget(command any, contents ...Widget) mouseTarget {
// 	return mouseTarget{
// 		command:  command,
// 		contents: contents,
// 	}
// }

// func (v *view) drawMouseTarget(content mouseTarget) {
// 	currentCommand := v.mouseTarget
// 	v.mouseTarget = content.command
// 	for _, child := range content.contents {
// 		v.draw(child)
// 	}
// 	v.mouseTarget = currentCommand
// }

// func Fixed(width int) Field {
// 	return Field{Width: width}
// }

// func Flex(flex int) Field {
// 	return Field{Flex: flex, Width: 4}
// }

// func Spacer(flex int) Field {
// 	return Field{Flex: flex, Width: 0}
// }

// func Pad(text string) Field {
// 	return Field{Text: text}
// }

// func Layout(fields Fields, contents ...Widget) layout {
// 	return layout{fields, contents}
// }

// type progressBar float64

// func ProgressBar(value float64) progressBar {
// 	return progressBar(value)
// }

// func (pb progressBar) runes(width int) []rune {
// 	result := make([]rune, width)
// 	progress := int(math.Round(float64(width*8) * float64(pb)))
// 	idx := 0
// 	for ; idx < progress/8; idx++ {
// 		result[idx] = '█'
// 	}
// 	if progress%8 > 0 {
// 		result[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
// 		idx++
// 	}
// 	for ; idx < width; idx++ {
// 		result[idx] = ' '
// 	}
// 	return result
// }
