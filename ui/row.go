package ui

import "math"

type row struct {
	widgets []Widget
}

func Row(ws ...Widget) Widget {
	return row{ws}
}

func (r row) Constraints() Constraints {
	width, flex := X(0), Flex(0)
	for _, widget := range r.widgets {
		c := widget.Constraints()
		width += c.Width.Size
		flex += c.Width.Flex
	}
	return MakeConstraints(width, flex, 1, 0)
}

func (r row) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
	sizes := make([]Constraint[X], len(r.widgets))
	for i, widget := range r.widgets {
		sizes[i] = widget.Constraints().Width
	}
	widths := calcSizes(width, sizes)
	for i, widget := range r.widgets {
		widget.Render(renderer, x, y, widths[i], height, style)
		x += widths[i]
	}
}

func calcSizes[S X | Y](size S, constraints []Constraint[S]) []S {
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
