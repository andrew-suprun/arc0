package controller

import (
	m "arch/model"
	w "arch/widgets"
	"time"
)

type controller struct {
	roots  []m.Root
	origin m.Root

	pendingHashes   map[m.Hash]struct{}
	duplicateHashes map[m.Hash]struct{}
	absentHashes    map[m.Hash]struct{}
	archives        map[m.Root]*archive
	folders         map[m.Path]*folder

	screenSize         m.ScreenSize
	lastMouseEventTime time.Time

	currentPath m.Path
	entries     []w.File

	feedback w.Feedback

	Errors []any

	quit bool
}

type archive struct {
	scanner       m.ArchiveScanner
	files         map[m.FullName]*w.File
	pending       map[m.FullName]*w.File
	progress      m.Progress
	progressState m.ProgressState
	totalSize     uint64
	totalHashed   uint64
	copySize      uint64
	totalCopied   uint64
}

type folder struct {
	selectedId    m.FileId
	offsetIdx     int
	sortColumn    w.SortColumn
	sortAscending []bool
}

func Run(fs m.FS, renderer w.Renderer, events m.EventChan, roots []m.Root) {
	rootFolder := &folder{
		sortAscending: []bool{true, false, false, false},
	}
	c := &controller{
		roots:  roots,
		origin: roots[0],

		archives:        map[m.Root]*archive{},
		folders:         map[m.Path]*folder{"": rootFolder},
		pendingHashes:   map[m.Hash]struct{}{},
		duplicateHashes: map[m.Hash]struct{}{},
		absentHashes:    map[m.Hash]struct{}{},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner: scanner,
			files:   map[m.FullName]*w.File{},
			pending: map[m.FullName]*w.File{},
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

		widget, feedback := screen.View()
		widget.Render(renderer, w.Position{X: 0, Y: 0}, w.Size(c.screenSize))
		c.feedback = feedback
		renderer.Show()
	}
}

func (a *archive) fileByFullName(name m.FullName) *w.File {
	return a.files[name]
}

func (a *archive) fileByNewName(name m.FullName) *w.File {
	if result, ok := a.pending[name]; ok {
		return result
	}
	result := a.files[name]
	if result != nil && result.Status != w.Pending {
		return result
	}
	return nil
}
