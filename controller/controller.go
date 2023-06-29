package controller

import (
	"arch/actor"
	"arch/model"
	"arch/widgets"
	"time"
)

type controller struct {
	fs          model.FS
	events      model.EventChan
	renderer    widgets.Renderer
	fileHandler actor.Actor[model.HandleFiles]

	roots              []string
	archives           map[string]*archive
	bySize             map[uint64][]*model.File
	byHash             map[string][]*model.File
	folders            map[string]*folder
	currentPath        string
	copySize           uint64
	fileCopied         uint64
	totalCopied        uint64
	pendingFiles       int
	duplicateFiles     int
	absentFiles        int
	screenSize         model.ScreenSize
	fileTreeLines      int
	lastMouseEventTime time.Time

	Errors []any

	quit bool
}

type archive struct {
	scanner   model.ArchiveScanner
	progress  model.ScanProgress
	totalSize uint64
	byName    map[string]*model.File
}

type folder struct {
	info          *model.File
	selected      *model.File
	lineOffset    int
	sortColumn    sortColumn
	sortAscending []bool
	entries       []*model.File
}

func Run(fs model.FS, renderer widgets.Renderer, ev model.EventChan, roots []string) {
	rootFolder := &folder{
		info:          &model.File{Kind: model.FileFolder},
		sortAscending: []bool{true, false, false, false},
	}
	c := &controller{
		fs:       fs,
		renderer: renderer,
		events:   ev,
		roots:    roots,
		archives: map[string]*archive{},
		bySize:   map[uint64][]*model.File{},
		byHash:   map[string][]*model.File{},
		folders:  map[string]*folder{"": rootFolder},
	}
	for _, path := range roots {
		scanner := fs.NewArchiveScanner(path)
		c.archives[path] = &archive{
			scanner: scanner,
			byName:  map[string]*model.File{},
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

func (c *controller) hashStatus(hash string, status model.ResulutionStatus) {
	for _, file := range c.byHash[hash] {
		file.Status = status
		c.updateFolderStatus(dir(file.Name))
	}
}

func (c *controller) ScreenSize() model.ScreenSize {
	return c.screenSize
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
