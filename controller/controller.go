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
	presence map[m.Hash]w.Presence

	screenSize         m.ScreenSize
	lastMouseEventTime time.Time

	currentPath m.Path
	entries     []w.File

	feedback w.Feedback

	Errors []any

	quit bool
}

func (c *controller) update(proc func(file *w.File)) {
	for _, archive := range c.archives {
		archive.update(proc)
	}
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
		presence: map[m.Hash]w.Presence{},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner: scanner,
			files:   map[m.Name]*w.File{},
			pending: map[m.Name]*w.File{},
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
