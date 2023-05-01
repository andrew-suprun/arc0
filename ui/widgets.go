package ui

import (
	"math"
)

type Widget interface {
	Size() Size
	Render(position Position, size Size, attributes *Attributes) Segments
}

type SizeKind int

const (
	Width SizeKind = iota
	Height
	Flex
)

type Size struct {
	Kind  SizeKind
	Value int
}

// *** Text ***
type text struct {
	runes []rune
	size  Size
}

func FlexText(txt string, flex int) text {
	return text{[]rune(txt), Size{Flex, flex}}
}

func Text(txt string) text {
	runes := []rune(txt)
	return text{runes, Size{Width, len(runes)}}
}

func (t text) Size() Size {
	return t.size
}

func (t text) Render(position Position, size Size, attributes *Attributes) Segments {
	if size.Value < 1 {
		return nil
	}
	if len(t.runes) > size.Value {
		t.runes = append(t.runes[:size.Value-1], '…')
	}
	diff := size.Value - len(t.runes)
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
	return Size{Width, math.MaxInt}
}

func (pb progressBar) Flex() int {
	return 2
}

func (pb progressBar) Render(position Position, size Size, attributes *Attributes) Segments {
	if size.Value < 1 {
		return nil
	}

	runes := make([]rune, size.Value)
	progress := int(math.Round(float64(size.Value*8) * float64(pb)))
	idx := 0
	for ; idx < progress/8; idx++ {
		runes[idx] = '█'
	}
	if progress%8 > 0 {
		runes[idx] = []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8]
		idx++
	}
	for ; idx < size.Value; idx++ {
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
	width, flex := 0, 0
	for _, w := range r.widgets {
		s := w.Size()
		if s.Kind == Flex && flex < s.Value {
			flex = s.Value
		} else if s.Kind == Width {
			width += w.Size().Value
		}
	}
	if flex > 0 {
		return Size{Flex, flex}
	}
	return Size{Width, width}
}

func (r row) Render(position Position, size Size, attributes *Attributes) Segments {
	result := Segments{}
	widths := calcSizes(size.Value, r.widgets)
	for i, w := range r.widgets {
		result = append(result, w.Render(position, Size{Width, widths[i]}, attributes)...)
		position.X += widths[i]
	}

	return result
}

func calcSizes(size int, widgets []Widget) []int {
	result := make([]int, len(widgets))
	sizes := make([]Size, len(widgets))
	for i, w := range widgets {
		sizes[i] = w.Size()
	}

	totalSize, fixedFields, totalFlex := 0, 0, 0
	for i, widgetSize := range sizes {
		if widgetSize.Kind != Flex {
			totalSize += widgetSize.Value
			fixedFields++
			result[i] = widgetSize.Value
		} else {
			totalFlex += widgetSize.Value
		}
	}
	if totalSize >= size {
		rate := float64(size) / float64(totalSize)
		for i, widgetSize := range sizes {
			if widgetSize.Kind != Flex {
				newSize := int(float64(result[i]) * rate)
				totalSize += newSize - result[i]
				result[i] = newSize
			} else {
				result[i] = 0
			}
		}
		for i, widgetSize := range sizes {
			if totalSize == size {
				break
			}
			if widgetSize.Kind != Flex {
				result[i]++
				totalSize++
			}
		}
	} else {
		rate := float64(size-totalSize) / float64(totalFlex)
		for i, widgetSize := range sizes {
			if widgetSize.Kind == Flex {
				newWidth := int(rate * float64(widgetSize.Value))
				totalSize += newWidth - result[i]
				result[i] = newWidth
			}
		}
		for i, widgetSize := range sizes {
			if totalSize == size {
				break
			}
			if widgetSize.Kind == Flex {
				result[i]++
				totalSize++
			}
		}
	}

	newWidth := 0
	for i := range result {
		newWidth += result[i]
	}

	return result
}

// *** column ***
type column struct {
	widgets []Widget
}

func Column(widgets ...Widget) Widget {
	return column{widgets}
}

func (c column) Size() Size {
	height, flex := 0, 0
	for _, w := range c.widgets {
		s := w.Size()
		if s.Kind == Flex && flex < s.Value {
			flex = s.Value
		} else if s.Kind == Height {
			height += w.Size().Value
		}
	}
	if flex > 0 {
		return Size{Flex, flex}
	}
	return Size{Height, height}
}

func (c column) Render(position Position, size Size, attributes *Attributes) Segments {
	result := Segments{}
	heights := calcSizes(size.Value, c.widgets)
	for i, w := range c.widgets {
		result = append(result, w.Render(position, Size{Width, heights[i]}, attributes)...)
		position.Y += heights[i]
	}

	return result
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
