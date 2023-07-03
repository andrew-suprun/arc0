package controller

import (
	m "arch/model"
	"arch/widgets"
	"time"
)

type controller struct {
	fs       m.FS
	events   m.EventChan
	renderer widgets.Renderer

	roots              []m.Root
	archives           map[m.Root]*archive
	bySize             map[uint64][]*m.File
	byHash             map[m.Hash][]*m.File
	folders            map[m.Path]*folder
	conflicts          map[m.FullName]struct{}
	currentPath        m.Path
	copySize           uint64
	fileCopied         uint64
	totalCopied        uint64
	pendingFiles       int
	duplicateFiles     int
	absentFiles        int
	screenSize         m.ScreenSize
	fileTreeLines      int
	lastMouseEventTime time.Time

	Errors []any

	quit bool
}

type archive struct {
	scanner   m.ArchiveScanner
	progress  m.ScanProgress
	totalSize uint64
	byName    map[m.FullName]*m.File
}

type folder struct {
	info          *m.File
	selected      *m.File
	selectedIdx   int
	offsetIdx     int
	sortColumn    sortColumn
	sortAscending []bool
	entries       []*m.File
}

func Run(fs m.FS, renderer widgets.Renderer, ev m.EventChan, roots []m.Root) {
	rootFolder := &folder{
		info:          &m.File{FileKind: m.FileFolder},
		sortAscending: []bool{true, false, false, false},
	}
	c := &controller{
		fs:        fs,
		renderer:  renderer,
		events:    ev,
		roots:     roots,
		archives:  map[m.Root]*archive{},
		bySize:    map[uint64][]*m.File{},
		byHash:    map[m.Hash][]*m.File{},
		folders:   map[m.Path]*folder{"": rootFolder},
		conflicts: map[m.FullName]struct{}{},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner: scanner,
			byName:  map[m.FullName]*m.File{},
		}
		scanner.ScanArchive()
	}

	for !c.quit {
		event := <-c.events
		c.handleEvent(event)
		select {
		case event = <-c.events:
			c.handleEvent(event)
		default:
		}

		c.folders[c.currentPath].sort()
		c.renderer.Reset()
		c.view().Render(c.renderer, widgets.Position{X: 0, Y: 0}, widgets.Size(c.ScreenSize()))
		c.renderer.Show()
	}
}

func (c *controller) hashStatus(hash m.Hash, status m.Status) {
	for _, file := range c.byHash[hash] {
		file.Status = status
		c.updateFolderStatus(file.Path)
	}
}

func (c *controller) ScreenSize() m.ScreenSize {
	return c.screenSize
}

type selectFile *m.File

type selectFolder *m.File

type sortColumn int

const (
	sortByName sortColumn = iota
	sortByStatus
	sortByTime
	sortBySize
)
