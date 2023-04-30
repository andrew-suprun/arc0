package ui

type Renderer interface {
	PollEvent() any
	Render(view ...Segment)
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

type Position struct {
	X, Y int
}

type Segment struct {
	Position   Position
	Runes      []rune
	Attributes *Attributes
}

type Segments []Segment

type Attributes struct {
	style        Style
	mouseTarget  any
	scrollTarget any
}

func DefaultAttributes() *Attributes {
	return &Attributes{}
}

func (a *Attributes) Style(style Style) *Attributes {
	result := *a
	result.style = style
	return &result
}

func (a *Attributes) MouseTarget(command any) *Attributes {
	result := *a
	result.mouseTarget = command
	return &result
}

func (a *Attributes) ScrollTarget(command any) *Attributes {
	result := *a
	result.scrollTarget = command
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
