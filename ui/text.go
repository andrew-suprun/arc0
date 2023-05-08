package ui

type text struct {
	runes       []rune
	constraints Constraints
}

func Text(txt string, width W, flex Flex) text {
	runes := []rune(txt)
	return text{runes, MakeConstraints(width, flex, 1, 0)}
}

func (t text) Constraints() Constraints {
	return t.constraints
}

func (t text) Render(renderer Renderer, x X, y Y, width W, _ H, style Style) {
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

	renderer.Text(t.runes, x, y, style)
}
