package ui

import (
	"log"
	"math"
)

type Widget interface {
	Constraints() Constraints
	Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes)
}

// *** Text ***
type text struct {
	runes       []rune
	constraints Constraints
}

func Text(txt string) text {
	runes := []rune(txt)
	return text{runes, MakeConstraints(W(len(runes)), 0, H(1), 0)}
}

func FixedText(txt string, width W, flex Flex) text {
	runes := []rune(txt)
	return text{runes, MakeConstraints(W(width), 0, H(1), 0)}
}

func FlexText(txt string, flex Flex) text {
	runes := []rune(txt)
	return text{runes, MakeConstraints(W(len(runes)), flex, H(1), 0)}
}

func (t text) Constraints() Constraints {
	return t.constraints
}

func (t text) Render(renderer Renderer, x X, y Y, width W, _ H, attributes *Attributes) {
	if width < 1 {
		return
	}
	if len(t.runes) > int(width) {
		t.runes = append(t.runes[:width-1], '…')
	}
	diff := int(width) - len(t.runes)
	for diff > 0 {
		t.runes = append(t.runes, ' ')
		diff--
	}

	renderer.Write(t.runes, x, y, attributes)
}

type progressBar struct {
	value float64
	width W
	flex  Flex
}

func ProgressBar(value float64, width W, flex Flex) progressBar {
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

func (pb progressBar) Render(renderer Renderer, x X, y Y, width W, _ H, attributes *Attributes) {
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

	renderer.Write(runes, x, y, attributes)
}

// *** Styled ***
type styled struct {
	style  Style
	widget Widget
}

func Styled(style Style, widget Widget) Widget {
	return styled{style: style, widget: widget}
}

func (s styled) Constraints() Constraints {
	return s.widget.Constraints()
}

func (s styled) Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes) {
	s.widget.Render(renderer, x, y, width, height, attributes.WithStyle(s.style))
}

// *** row ***
type row struct {
	widgets []Widget
}

func Row(ws ...Widget) Widget {
	return row{ws}
}

func (r row) Constraints() Constraints {
	width, flex := W(0), Flex(0)
	for _, widget := range r.widgets {
		c := widget.Constraints()
		width += c.Width.Size
		flex += c.Width.Flex
	}
	return MakeConstraints(width, flex, 1, 0)
}

func (r row) Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes) {
	sizes := make([]Constraint[W], len(r.widgets))
	for i, widget := range r.widgets {
		sizes[i] = widget.Constraints().Width
	}
	widths := calcSizes(width, sizes)
	for i, widget := range r.widgets {
		widget.Render(renderer, x, y, widths[i], height, attributes)
		x = x.Inc(widths[i])
	}
}

func calcSizes[S ~int](size S, constraints []Constraint[S]) []S {
	result := make([]S, len(constraints))
	totalSize, totalFlex := S(0), Flex(0)
	for i, constraint := range constraints {
		result[i] = constraint.Size
		totalSize += constraint.Size
		totalFlex += constraint.Flex
	}
	for totalSize > size {
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

	if totalSize < size {
		diff := size - totalSize
		remainders := make([]float64, len(constraints))
		for i, constraint := range constraints {
			rate := float64(Flex(diff)*constraint.Flex) / float64(totalFlex)
			remainders[i] = rate - math.Floor(rate)
			result[i] += S(rate)
		}
		totalSize := S(0)
		for _, size := range result {
			totalSize += size
		}
		for i := range result {
			if totalSize == size {
				break
			}
			if constraints[i].Flex > 0 {
				result[i]++
				totalSize++
			}
		}
		for i := range result {
			if totalSize == size {
				break
			}
			if constraints[i].Flex == 0 {
				result[i]++
				totalSize++
			}
		}
	}

	return result
}

// *** column ***
type column struct {
	constraint Constraint[H]
	widgets    []Widget
}

func Column(flex Flex, widgets ...Widget) Widget {
	height := H(0)
	for _, widget := range widgets {
		height += widget.Constraints().Height.Size
	}
	return column{Constraint[H]{height, flex}, widgets}
}

func (c column) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: c.constraint}
}

func (c column) Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes) {
	sizes := make([]Constraint[H], len(c.widgets))
	for i, widget := range c.widgets {
		sizes[i] = widget.Constraints().Height
	}
	heights := calcSizes(height, sizes)
	for i, widget := range c.widgets {
		widget.Render(renderer, x, y, width, height, attributes)
		y = y.Inc(heights[i])
	}
}

// *** VSpacer ***
type VSpacer struct{}

func (w VSpacer) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 1}, Height: Constraint[H]{0, 1}}
}

func (w VSpacer) Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes) {
	log.Println("VSpacer", x, y, width, height)
	runes := make([]rune, width)
	for i := range runes {
		runes[i] = ' '
	}
	for i := 0; i < int(height); i++ {
		renderer.Write(runes, x, y+Y(i), attributes)
	}
}

// *** NullWidget ***
type NullWidget struct{}

func (w NullWidget) Constraints() Constraints {
	return Constraints{Width: Constraint[W]{0, 0}, Height: Constraint[H]{0, 0}}
}

func (w NullWidget) Render(renderer Renderer, x X, y Y, width W, height H, attributes *Attributes) {
}
