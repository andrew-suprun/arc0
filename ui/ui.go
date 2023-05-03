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
	Write(runes []rune, x X, y Y, attributes *Attributes)
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

type Attributes struct {
	Style        Style
	MouseTarget  any
	ScrollTarget any
}

func DefaultAttributes() *Attributes {
	return &Attributes{}
}

func (a *Attributes) WithStyle(style Style) *Attributes {
	result := *a
	result.Style = style
	return &result
}

func (a *Attributes) WithMouseTarget(command any) *Attributes {
	result := *a
	result.MouseTarget = command
	return &result
}

func (a *Attributes) WithScrollTarget(command any) *Attributes {
	result := *a
	result.ScrollTarget = command
	return &result
}

type Style int

const (
	NoStyle Style = iota
	StyleDefault
	StyleHeader
	StyleAppTitle
	StyleArchiveName
	StyleFile
	StyleFolder
	StyleProgressBar
	StyleArchiveHeader
)
