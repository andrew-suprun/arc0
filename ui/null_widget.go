package ui

type NullWidget struct{}

func (w NullWidget) Constraint() Constraint {
	return Constraint{Size{0, 0}, Flex{0, 0}}
}

func (w NullWidget) Render(ctx *Context, pos Position, size Size) {}
