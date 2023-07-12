package controller

import (
	m "arch/model"
	w "arch/widgets"
	"time"
)

type controller struct {
	roots  []m.Root
	origin m.Root

	archives map[m.Root]*archive
	folders  map[m.Path]*folder
	files    map[m.Id]*w.File
	state    map[m.Hash]w.State

	copyingProgress m.CopyingProgress
	copySize        uint64
	totalCopied     uint64

	screenSize         m.ScreenSize
	lastMouseEventTime time.Time

	currentPath m.Path
	entries     []w.File

	feedback w.Feedback

	Errors []any

	quit bool
}

type archive struct {
	scanner         m.ArchiveScanner
	hashingProgress m.HashingProgress
	progressState   m.ProgressState
	totalSize       uint64
	totalHashed     uint64
}

type folder struct {
	selectedId    m.Id
	selectedIdx   int
	offsetIdx     int
	sortColumn    w.SortColumn
	sortAscending []bool
}

func Run(fs m.FS, renderer w.Renderer, events m.EventChan, roots []m.Root) {
	c := &controller{
		roots:  roots,
		origin: roots[0],

		archives: map[m.Root]*archive{},
		folders:  map[m.Path]*folder{},
		state:    map[m.Hash]w.State{},
		files:    map[m.Id]*w.File{},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner: scanner,
		}
		scanner.Send(m.ScanArchive{})
	}

	for !c.quit {
		event := <-events
		c.handleEvent(event)
		select {
		case event = <-events:
			c.handleEvent(event)
		default:
		}

		renderer.Reset()
		screen := c.buildScreen()

		widget := screen.View(&c.feedback)
		widget.Render(renderer, w.Position{X: 0, Y: 0}, w.Size(c.screenSize))
		renderer.Show()
	}
}

func (c *controller) currentFolder() *folder {
	curFolder, ok := c.folders[c.currentPath]
	if !ok {
		curFolder = &folder{
			sortAscending: []bool{true, false, false, false},
		}
		c.folders[c.currentPath] = curFolder
	}
	return curFolder
}

func (c *controller) getSelectedId() (result m.Id) {
	if len(c.entries) == 0 {
		return m.Id{}
	}
	folder := c.currentFolder()
	idx := 0
	for idx = range c.entries {
		if c.entries[idx].Id == folder.selectedId {
			return folder.selectedId
		}
	}
	if folder.selectedIdx >= len(c.entries) {
		folder.selectedIdx = len(c.entries) - 1
	}
	if folder.selectedIdx < 0 {
		folder.selectedIdx = 0
	}
	folder.selectedId = c.entries[folder.selectedIdx].Id
	return folder.selectedId
}

func (c *controller) setSelectedId(id m.Id) {
	folder := c.currentFolder()
	folder.selectedId = id
	for idx, entry := range c.entries {
		if entry.Id == id {
			folder.selectedIdx = idx
		}
	}
}

func (c *controller) getSelectedIdx() (result int) {
	return c.currentFolder().selectedIdx
}

func (c *controller) setSelectedIdx(idx int) {
	folder := c.currentFolder()
	if idx >= len(c.entries) {
		idx = len(c.entries) - 1
	}
	if idx < 0 {
		idx = 0
	}
	folder.selectedIdx = idx
	folder.selectedId = c.entries[idx].Id
}
