package controller

import (
	"arch/actor"
	"arch/model"
	"arch/widgets"
	"time"
)

type controller struct {
	fs       model.FS
	events   model.EventChan
	renderer widgets.Renderer

	roots              []string
	archives           map[string]*archive
	bySize             map[uint64][]*model.File
	byHash             map[string][]*model.File
	folders            map[string]*folder
	currentPath        string
	screenSize         model.ScreenSize
	fileTreeLines      int
	lastMouseEventTime time.Time

	Errors []any

	quit bool
}

type archive struct {
	scanner     actor.Actor[model.Msg]
	progress    model.Progress
	totalSize   uint64
	copySize    uint64
	totalCopied uint64
	byINode     map[uint64]*model.File
}

type folder struct {
	info          *model.File
	selected      *model.File
	lineOffset    int
	sortColumn    sortColumn
	sortAscending []bool
	entries       []*model.File
}

func Run(fs model.FS, renderer widgets.Renderer, ev model.EventChan, paths []string) {
	rootFolder := &folder{
		info:          &model.File{Kind: model.FileFolder},
		sortAscending: []bool{true, false, false, false},
	}
	m := &controller{
		fs:       fs,
		renderer: renderer,
		events:   ev,
		roots:    paths,
		archives: map[string]*archive{},
		bySize:   map[uint64][]*model.File{},
		byHash:   map[string][]*model.File{},
		folders:  map[string]*folder{"": rootFolder},
	}
	for _, path := range paths {
		s := fs.NewScanner(path)
		m.archives[path] = &archive{
			scanner: actor.NewActor(s.Handler),
			byINode: map[uint64]*model.File{},
		}
	}

	for _, archive := range m.archives {
		archive.scanner.Send(model.ScanArchive{})
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

		m.folders[m.currentPath].sort()
		m.renderer.Reset()
		m.view().Render(m.renderer, widgets.Position{X: 0, Y: 0}, widgets.Size(m.ScreenSize()))
		m.renderer.Show()
	}
}

func (m *controller) ScreenSize() model.ScreenSize {
	return m.screenSize
}

type selectFile *model.File

type selectFolder *model.File

type sortColumn int

const (
	sortByName sortColumn = iota
	sortByStatus
	sortByTime
	sortBySize
)
