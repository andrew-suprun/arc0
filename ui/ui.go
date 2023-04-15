package ui

type Renderer interface {
	PollEvent() any
	Render(screen Screen)
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

type Screen []Segment

type Segment struct {
	X, Y        int
	Runes       []rune
	Style       Style
	MouseTarget any
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
