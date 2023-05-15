package device

type Device interface {
	PollEvent() any
	Text(runes []rune, x, y int, style Style)
	Show()
	Sync()
	Exit()
}

type Screen [][]Char

type Char struct {
	Rune  rune
	Style Style
}

type Style struct {
	FG, BG byte
	Flags  Flags
}

type Flags byte

const (
	Bold    Flags = 1
	Italic  Flags = 2
	Reverse Flags = 4
)

type RenderMode bool

const (
	Regular RenderMode = false
	Sync    RenderMode = true
)
