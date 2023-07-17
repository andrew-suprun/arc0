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
	ScreenSize     m.ScreenSize
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
	FileTreeLines  int
	FPS            int
}

func (s *Screen) String() string {
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "Screen{")
	fmt.Fprintf(buf, "  ScreenSize:     {Width: %d, Height %d},\n", s.ScreenSize.Width, s.ScreenSize.Height)
	fmt.Fprintf(buf, "  CurrentPath:    %q,\n", s.CurrentPath)
	fmt.Fprintf(buf, "  SelectedId:     %q,\n", s.SelectedId)
	fmt.Fprintf(buf, "  OffsetIdx:      %d,\n", s.OffsetIdx)
	fmt.Fprintf(buf, "  SortColumn:     %s,\n", s.SortColumn)
	fmt.Fprintf(buf, "  SortAscending:  %v,\n", s.SortAscending)
	fmt.Fprintf(buf, "  PendingFiles:   %d,\n", s.PendingFiles)
	fmt.Fprintf(buf, "  DuplicateFiles: %d,\n", s.DuplicateFiles)
	fmt.Fprintf(buf, "  AbsentFiles:    %d,\n", s.AbsentFiles)
	fmt.Fprintf(buf, "  FileTreeLines:  %d,\n", s.FileTreeLines)
	if len(s.Entries) > 0 {
		fmt.Fprintf(buf, "  Entries: {\n")
		for _, entry := range s.Entries {
			fmt.Fprintf(buf, "    %s:\n", &entry.File)
		}
		fmt.Fprintf(buf, "  }\n")
	}
	if len(s.Progress) > 0 {
		fmt.Fprintf(buf, "  Progress: {\n")
		for _, progress := range s.Progress {
			fmt.Fprintf(buf, "    {Root: %q, Tab: %q, Value: %f}:\n", progress.Root, progress.Tab, progress.Value)
		}
		fmt.Fprintf(buf, "  }\n")
	}
	return buf.String()
}

type SortColumn int

const (
	SortByName SortColumn = iota
	SortByTime
	SortBySize
)

func (c SortColumn) String() string {
	switch c {
	case SortByName:
		return "SortByName"
	case SortByTime:
		return "SortByTime"
	case SortBySize:
		return "SortBySize"
	}
	return "Illegal Sort Solumn"
}

type File struct {
	m.File
	Kind
	State
}

type Kind int

const (
	FileRegular Kind = iota
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
	Root  m.Root
	Tab   string
	Value float64
}

func (f *File) String() string {
	return fmt.Sprintf("File{FileId: %q, Kind: %s, Size: %d, Hash: %q, State: %s}", f.Id, f.Kind, f.Size, f.Hash, f.State)
}

func (k Kind) String() string {
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
