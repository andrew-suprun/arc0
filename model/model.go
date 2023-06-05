package model

import (
	"arch/events"
	"arch/files"
	"arch/widgets"
	"fmt"
	"time"
)

type model struct {
	fs       files.FS
	events   events.EventChan
	renderer widgets.Renderer

	archives           []*archive
	byHash             map[string][]*File
	folders            map[string]*folder
	breadcrumbs        []*folder
	screenSize         ScreenSize
	fileTreeLines      int
	lastMouseEventTime time.Time

	Errors []any

	quit bool
}

type archive struct {
	archivePath string
	scanner     files.Scanner
	scanState   events.ScanProgress
	byIno       map[uint64]*File
}

type folder struct {
	info          *File
	selected      *File
	lineOffset    int
	sortColumn    sortColumn
	sortAscending []bool
	entries       []*File
}

func Run(fs files.FS, renderer widgets.Renderer, ev events.EventChan, paths []string) {
	rootFolder := &folder{
		info:          &File{FileMeta: events.FileMeta{Name: "Root"}, Kind: FileFolder},
		sortAscending: []bool{true, false, false, false},
	}
	m := &model{
		fs:          fs,
		renderer:    renderer,
		events:      ev,
		archives:    make([]*archive, len(paths)),
		byHash:      map[string][]*File{},
		folders:     map[string]*folder{"": rootFolder},
		breadcrumbs: []*folder{rootFolder},
	}
	for i, path := range paths {
		m.archives[i] = &archive{
			archivePath: path,
			scanner:     fs.NewScanner(path),
			byIno:       map[uint64]*File{},
		}
	}

	for _, archive := range m.archives {
		archive.scanner.ScanArchive()
	}

	for !m.quit {
		event := <-m.events
		m.handleEvent(event)
		events := 0
		select {
		case event = <-m.events:
			m.handleEvent(event)
			events++
		default:
		}

		m.renderer.Reset()
		m.view().Render(m.renderer, widgets.Position{X: 0, Y: 0}, widgets.Size(m.ScreenSize()))
		m.renderer.Show()
	}
}

func (m *model) currentFolder() *folder {
	return m.breadcrumbs[len(m.breadcrumbs)-1]
}

func (m *model) ScreenSize() ScreenSize {
	return m.screenSize
}

type ScreenSize struct {
	Width, Height int
}

type File struct {
	events.FileMeta
	Kind   FileKind
	Status FileStatus
	Hash   string
}

func (f File) String() string {
	return fmt.Sprintf("File{Meta: %s, Kind: %s, Status: %s}", f.FileMeta.String(), f.Kind, f.Status)
}

type Files []*File

type FileKind int

const (
	FileRegular FileKind = iota
	FileFolder
)

func (k FileKind) String() string {
	switch k {
	case FileFolder:
		return "FileFolder"
	case FileRegular:
		return "FileRegular"
	}
	return "UNKNOWN FILE KIND"
}

type FileStatus int

const (
	Identical FileStatus = iota
	SourceOnly
	CopyOnly
)

func (s FileStatus) String() string {
	switch s {
	case Identical:
		return "Identical"
	case SourceOnly:
		return "SourceOnly"
	case CopyOnly:
		return "CopyOnly"
	}
	return "UNKNOWN FILE KIND"
}

func (s FileStatus) Merge(other FileStatus) FileStatus {
	if s > other {
		return s
	}
	return other
}

type selectFile *File

type selectFolder *File

type sortColumn int

const (
	sortByName sortColumn = iota
	sortByStatus
	sortByTime
	sortBySize
)
