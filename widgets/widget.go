package widgets

import (
	m "arch/model"
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
	AddMouseTarget(m.MouseTarget, Position, Size)
	AddScrollArea(m.Scroll, Position, Size)
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

type Position struct {
	X, Y int
}

//lint:ignore U1000 Casted into m.ScreenSize
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

type Screen struct {
	CurrentPath    m.Path
	Entries        []*File
	Progress       []ProgressInfo
	SelectedId     m.Id
	OffsetIdx      int
	SortColumn     SortColumn
	SortAscending  []bool
	PendingFiles   int
	DuplicateFiles int
	AbsentFiles    int
}

type Feedback struct {
	Entries       int // TODO Remove entries from controller
	FileTreeLines int
}

type SortColumn int

const (
	SortByName SortColumn = iota
	SortByTime
	SortBySize
)

type File struct {
	m.FileMeta
	FileKind
	m.Hash
	State
}

type FileKind int

const (
	FileRegular FileKind = iota
	FileFolder
)

type State int

const (
	Resolved State = iota
	Pending
	Duplicate
	Absent
)

type ProgressInfo struct {
	m.Root
	Tab   string
	Value float64
}

func (f *File) String() string {
	return fmt.Sprintf("File{FileId: %q, Kind: %s, Size: %d, Hash: %q}", f.Id, f.FileKind, f.Size, f.Hash)
}

func (k FileKind) String() string {
	switch k {
	case FileFolder:
		return "FileFolder"
	case FileRegular:
		return "FileRegular"
	}
	return "UNKNOWN FILE KIND"
}

func (p State) String() string {
	switch p {
	case Resolved:
		return "Resolved"
	case Pending:
		return "Pending"
	case Duplicate:
		return "Duplicate"
	case Absent:
		return "Absent"
	}
	return "UNKNOWN FILE STATE"
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
