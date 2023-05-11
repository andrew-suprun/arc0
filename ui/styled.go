package ui

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

func (s styled) Render(renderer Renderer, x X, y Y, width X, height Y, style Style) {
	s.widget.Render(renderer, x, y, width, height, s.style)
}
