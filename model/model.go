package model

import (
	"path/filepath"
	"time"
)

type Model struct {
	Archives []Archive

	Root          *FileInfo
	Breadcrumbs   []Folder
	ScreenSize    Size
	SortColumn    SortColumn
	SortAscending []bool
	FileTreeLines int

	Errors []any

	Quit bool
}

func NewModel(paths ...string) *Model {
	m := &Model{
		Archives: make([]Archive, len(paths)),
	}
	for i := range paths {
		m.Archives[i].Path = paths[i]
	}
	return m
}

func (m *Model) CurerntFolder() *Folder {
	if len(m.Breadcrumbs) == 0 {
		return nil
	}
	return &m.Breadcrumbs[len(m.Breadcrumbs)-1]
}

type Archive struct {
	Path      string
	ScanState *ScanState
	Files     FileMetas
}

type ScanState struct {
	Path      string
	Name      string
	Remaining time.Duration
	Progress  float64
}

type FileMeta struct {
	Archive  string
	FullName string
	Size     int
	ModTime  time.Time
	Hash     string
}

type FileMetas []*FileMeta

type FileInfo struct {
	*FileMeta
	Name   string
	Kind   FileKind
	Status FileStatus
	Files  FileInfos
}

func (f *FileInfo) AbsName() string {
	return filepath.Join(f.Archive, f.FullName)
}

type FileInfos []*FileInfo

type FileKind int

const (
	FileRegular FileKind = iota
	FileFolder
)

type FileStatus int

const (
	Identical FileStatus = iota
	SourceOnly
	CopyOnly
)

func (s FileStatus) Merge(other FileStatus) FileStatus {
	if s > other {
		return s
	}
	return other
}

type Folder struct {
	File       *FileInfo
	Selected   *FileInfo
	LineOffset int
}

type ScanError struct {
	Archive string
	Path    string
	Error   error
}

type Size struct {
	Width, Height int
}

type Position struct {
	X int
	Y int
}

type MouseTargetArea struct {
	Command any
	Pos     Position
	Size    Size
}

type ScrollArea struct {
	Command any
	Pos     Position
	Size    Size
}

// Commands:

type SelectFile *FileInfo

type SelectFolder *FileInfo

type SortColumn int

const (
	SortByName SortColumn = iota
	SortByStatus
	SortByTime
	SortBySize
)

type EventChan chan Event

type Event interface {
	HandleEvent(m *Model)
}

type View interface {
	View(model *Model)
}

type FS interface {
	Scan(path string) error
}
