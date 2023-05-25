package device

type Device interface {
	AddMouseTarget(cmd any, pos Position, size Size)
	AddScrollArea(cmd any, pos Position, size Size)
	SetStyle(style Style)
	CurrentStyle() Style
	Text(runes []rune, pos Position)
	Show()
	Reset()
}

type Position struct {
	X int
	Y int
}

type Size struct {
	Width  int
	Height int
}

type Flex struct {
	X int
	Y int
}

type Constraint struct {
	Size
	Flex
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
