package widgets

import (
	"arch/events"
	"fmt"
	"strings"
)

type Widget interface {
	Constraint() Constraint
	Render(Renderer, Position, Size)
	String() string
	ToString(*strings.Builder, string)
}

type Renderer interface {
	AddMouseTarget(events.MouseTarget, Position, Size)
	AddScrollArea(events.Scroll, Position, Size)
	SetStyle(style Style)
	CurrentStyle() Style
	Text([]rune, Position)
	Reset()
	Show()
}

type Constraint struct {
	Size
	Flex
}

func (c Constraint) String() string {
	return fmt.Sprintf("Constraint(Size(Width: %d, Height: %d), Flex(X: %d, Y:%d))", c.Width, c.Height, c.X, c.Y)
}

type Position struct {
	X, Y int
}

type Size struct {
	Width, Height int
}

type Flex struct {
	X, Y int
}

type Style struct {
	FG, BG byte
	Flags  Flags
}

func (s Style) String() string {
	return fmt.Sprintf("Style{FG: %d, BG: %d, Flags: {%s}", s.FG, s.BG, s.Flags)
}

type Flags byte

const (
	Bold    Flags = 1
	Italic  Flags = 2
	Reverse Flags = 4
)

func (f Flags) String() string {
	flags := []string{}
	if f&Bold == Bold {
		flags = append(flags, "Bold")
	}
	if f&Italic == Italic {
		flags = append(flags, "Italic")
	}
	if f&Reverse == Reverse {
		flags = append(flags, "Reverse")
	}
	return strings.Join(flags, ", ")
}

func toString[W Widget](w W) string {
	buf := &strings.Builder{}
	w.ToString(buf, "")
	return buf.String()
}
