package ui

type X int
type Y int
type Flex int

type Constraint[S X | Y] struct {
	Size S
	Flex Flex
}

type Constraints struct {
	Width  Constraint[X]
	Height Constraint[Y]
}

func MakeConstraints(width X, wFlex Flex, height Y, hFlex Flex) Constraints {
	return Constraints{Width: Constraint[X]{width, wFlex}, Height: Constraint[Y]{height, hFlex}}
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
