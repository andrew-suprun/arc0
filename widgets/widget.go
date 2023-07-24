package widgets

import (
	m "arch/model"
	"fmt"
	"strings"
	"time"
)

type Widget interface {
	Constraint() Constraint
	Render(*Screen, Position, Size)
	String() string
	ToString(*strings.Builder, string)
}

type Constraint struct {
	Size
	Flex
}

type Position struct {
	X, Y int
}

// XXX lint:ignore U1000 Casted into m.ScreenSize
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

type Flags byte

const (
	Bold    Flags = 1
	Italic  Flags = 2
	Reverse Flags = 4
)

type View struct {
	ScreenSize     m.ScreenSize
	CurrentPath    m.Path
	Entries        []*m.File
	Progress       []ProgressInfo
	SelectedId     m.Id
	OffsetIdx      int
	SortColumn     m.SortColumn
	SortAscending  []bool
	PendingFiles   int
	DuplicateFiles int
	AbsentFiles    int
	FileTreeLines  int
	FPS            int
}

type ProgressInfo struct {
	Root          m.Root
	Tab           string
	Value         float64
	Speed         float64
	TimeRemaining time.Duration
}

func (s Style) String() string {
	return fmt.Sprintf("Style{FG: %d, BG: %d, Flags: {%s}", s.FG, s.BG, s.Flags)
}

func (c Constraint) String() string {
	return fmt.Sprintf("Constraint(Size(Width: %d, Height: %d), Flex(X: %d, Y:%d))", c.Width, c.Height, c.X, c.Y)
}

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
