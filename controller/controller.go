package controller

import (
	m "arch/model"
	"arch/stream"
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
	totalCopiedSize uint64
	fileCopiedSize  uint64
	archivesScanned bool

	lastMouseEventTime time.Time
	currentPath        m.Path
	selectedIdx        int

	frames   int
	prevTick time.Time

	view w.View

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
	selectedId    m.Id
	offsetIdx     int
	sortColumn    w.SortColumn
	sortAscending []bool
}

func Run(fs m.FS, renderer w.Renderer, events stream.Stream[m.Event], roots []m.Root) {
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
		event := events.Pull()
		c.handleEvent(event)
		for _, event := range events.PullAll() {
			c.handleEvent(event)
		}

		c.frames++
		screen := w.NewScreen(c.view.ScreenSize)
		c.buildView().RootWidget().Render(screen, w.Position{X: 0, Y: 0}, w.Size(c.view.ScreenSize))
		renderer.Push(screen)
	}
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

func (c *controller) every(f func(entry *m.File)) {
	for _, entries := range c.files {
		for _, entry := range entries {
			f(entry)
		}
	}
}

func (c *controller) selectedEntry() *w.File {
	return c.entry(c.view.SelectedId)
}

func (c *controller) entry(id m.Id) *w.File {
	for _, entry := range c.view.Entries {
		if entry.Id == id {
			return entry
		}
	}
	return nil
}
