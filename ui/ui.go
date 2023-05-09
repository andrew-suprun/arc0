package ui

type X int
type Y int
type W int
type H int
type Flex int

func (x X) Inc(w W) X {
	return x + X(w)
}

func (y Y) Inc(h H) Y {
	return y + Y(h)
}

type Constraint[S ~int] struct {
	Size S
	Flex Flex
}

type Constraints struct {
	Width  Constraint[W]
	Height Constraint[H]
}

func MakeConstraints(width W, wFlex Flex, height H, hFlex Flex) Constraints {
	return Constraints{Width: Constraint[W]{width, wFlex}, Height: Constraint[H]{height, hFlex}}
}

type Renderer interface {
	PollEvent() any
	MouseTarget(command any, x X, y Y)
	ScrollTarget(command any, x X, y Y)
	Text(runes []rune, x X, y Y, style Style)
	Show()
	Sync()
	Exit()
}

type MouseEvent struct {
	Col, Line int
}

type KeyEvent struct {
	Name string
	Rune rune
}

type ResizeEvent struct {
	Width, Height int
}

type Style struct {
	FG, BG                int
	Bold, Italic, Reverse bool
}
