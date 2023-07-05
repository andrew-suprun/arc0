package controller

import (
	m "arch/model"
	w "arch/widgets"
	"time"
)

type controller struct {
	fs     m.FS
	events m.EventChan

	roots    []m.Root
	origin   m.Root
	archives map[m.Root]*archive
	folders  map[m.Path]*folder
	hashById map[m.FileId]m.Hash

	screenSize         m.ScreenSize
	lastMouseEventTime time.Time

	currentPath m.Path
	entries     []*w.File

	feedback w.Feedback

	Errors []any

	quit bool
}

type archive struct {
	scanner     m.ArchiveScanner
	infoByName  map[m.FullName]*w.File
	infosBySize map[uint64]map[*w.File]struct{}
	progress    m.ScanProgress
	totalSize   uint64
	copySize    uint64
	fileCopied  uint64
	totalCopied uint64
}

type folder struct {
	selectedId    m.FileId
	offsetIdx     int
	sortColumn    w.SortColumn
	sortAscending []bool
}

func Run(fs m.FS, renderer w.Renderer, ev m.EventChan, roots []m.Root) {
	rootFolder := &folder{
		sortAscending: []bool{true, false, false, false},
	}
	c := &controller{
		fs:       fs,
		events:   ev,
		roots:    roots,
		origin:   roots[0],
		archives: map[m.Root]*archive{},
		folders:  map[m.Path]*folder{"": rootFolder},
		hashById: map[m.FileId]m.Hash{},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner:     scanner,
			infosBySize: map[uint64]map[*w.File]struct{}{},
			infoByName:  map[m.FullName]*w.File{},
		}
		scanner.Send(m.ScanArchive{})
	}

	for !c.quit {
		event := <-c.events
		c.handleEvent(event)
		select {
		case event = <-c.events:
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
