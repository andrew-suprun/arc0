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
	selectedIdx        int

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
	selectedId    m.Id
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

func (c *controller) selectedEntry() *w.File {
	return c.entry(c.screen.SelectedId)
}

func (c *controller) entry(id m.Id) *w.File {
	for _, entry := range c.screen.Entries {
		if entry.Id == id {
			return entry
		}
	}
	return nil
}
