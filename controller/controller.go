package controller

import (
	m "arch/model"
	w "arch/widgets"
	"time"
)

type controller struct {
	roots  []m.Root
	origin m.Root

	archives        map[m.Root]*archive
	folders         map[m.Path]*folder
	files           map[m.Hash][]*m.File
	state           map[m.Hash]w.State
	copySize        uint64
	totalCopied     uint64
	fileCopied      uint64
	archivesScanned bool

	lastMouseEventTime time.Time
	currentPath        m.Path

	frames   int
	prevTick time.Time

	screen w.Screen

	Errors []any

	quit bool
}

type archive struct {
	scanner       m.ArchiveScanner
	progressState m.ProgressState
	totalSize     uint64
	totalHashed   uint64
	fileHashed    uint64
}

type folder struct {
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
		files:    map[m.Hash][]*m.File{},
		state:    map[m.Hash]w.State{},
	}

	go ticker(events)

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

		c.frames++
		renderer.Reset()
		c.buildScreen()

		widget := c.screen.View()
		widget.Render(renderer, w.Position{X: 0, Y: 0}, w.Size(c.screen.ScreenSize))
		renderer.Show()
	}

	go func() {
		for {
			if _, ok := <-events; !ok {
				break
			}
		}
	}()
}

func (c *controller) currentFolder() *folder {
	curFolder, ok := c.folders[c.currentPath]
	if !ok {
		curFolder = &folder{
			sortAscending: []bool{true, false, false},
		}
		c.folders[c.currentPath] = curFolder
	}
	return curFolder
}

func (c *controller) do(f func(entry *m.File) bool) {
	for _, entries := range c.files {
		for _, entry := range entries {
			if !f(entry) {
				return
			}
		}
	}
}

func (c *controller) find(f func(entry *m.File) bool) *m.File {
	for _, entries := range c.files {
		for _, entry := range entries {
			if f(entry) {
				return entry
			}
		}
	}
	return nil
}

func (c *controller) getSelectedFile() *w.File {
	selectedIdx := c.currentFolder().selectedIdx
	if selectedIdx >= len(c.screen.Entries) {
		return nil
	}
	return c.screen.Entries[selectedIdx]
}

func (c *controller) getSelectedIdx() int {
	return c.currentFolder().selectedIdx
}

func (c *controller) getSelectedId() m.Id {
	file := c.getSelectedFile()
	if file != nil {
		return file.Id
	}
	return m.Id{}
}

func (c *controller) setSelectedIdx(idx int) {
	if idx >= len(c.screen.Entries) {
		idx = len(c.screen.Entries) - 1
	}
	if idx < 0 {
		idx = 0
	}
	c.currentFolder().selectedIdx = idx
}

func (c *controller) setSelectedId(id m.Id) {
	for idx, entry := range c.screen.Entries {
		if entry.Id == id {
			c.currentFolder().selectedIdx = idx
			return
		}
	}
	c.currentFolder().selectedIdx = 0
}
