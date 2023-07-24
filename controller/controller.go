package controller

import (
	m "arch/model"
	"arch/stream"
	w "arch/widgets"
	"fmt"
	"strings"
	"time"
)

type controller struct {
	roots  []m.Root
	origin m.Root

	archives map[m.Root]*archive
	folders  map[m.Path]*folder
	byId     map[m.Id]*m.File
	bySize   map[uint64][]*m.File
	byHash   map[m.Hash][]*m.File

	copySize           uint64
	totalCopiedSize    uint64
	fileCopiedSize     uint64
	prevCopied         uint64
	copySpeed          float64
	timeRemaining      time.Duration
	lastMouseEventTime time.Time
	currentPath        m.Path
	screenSize         m.ScreenSize
	pendingFiles       int
	duplicateFiles     int
	absentFiles        int
	fileTreeLines      int
	progressInfos      []w.ProgressInfo

	frames   int
	fps      int
	prevTick time.Time

	Errors []any

	quit bool
}

type archive struct {
	scanner       m.ArchiveScanner
	progressState m.ProgressState
	totalSize     uint64
	totalHashed   uint64
	fileHashed    uint64
	prevHashed    uint64
	speed         float64
	timeRemaining time.Duration
}

type folder struct {
	entries       map[m.Base]*m.File
	selectedEntry *m.File
	offsetIdx     int
	sortColumn    m.SortColumn
	sortAscending []bool
}

func Run(fs m.FS, renderer w.Renderer, events *stream.Stream[m.Event], roots []m.Root) {
	c := &controller{
		roots:  roots,
		origin: roots[0],

		archives: map[m.Root]*archive{},
		folders:  map[m.Path]*folder{},
		byId:     map[m.Id]*m.File{},
		bySize:   map[uint64][]*m.File{},
		byHash:   map[m.Hash][]*m.File{},
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
		for _, event := range events.Pull() {
			c.handleEvent(event)
		}

		c.frames++
		screen := w.NewScreen(c.screenSize)
		c.RootWidget().Render(screen, w.Position{X: 0, Y: 0}, w.Size(c.screenSize))
		renderer.Push(screen)
	}
}

func (c *controller) currentFolder() *folder {
	return c.getFolder(c.currentPath)
}

func (c *controller) getFolder(path m.Path) *folder {
	pathFolder, ok := c.folders[path]
	if !ok {
		pathFolder = &folder{
			entries:       map[m.Base]*m.File{},
			sortAscending: []bool{true, false, false},
		}
		c.folders[path] = pathFolder
	}
	return pathFolder
}

func (c *controller) screenString() string {
	f := c.currentFolder()
	buf := &strings.Builder{}
	fmt.Fprintln(buf, "Screen{")
	fmt.Fprintf(buf, "  ScreenSize:     {Width: %d, Height %d},\n", c.screenSize.Width, c.screenSize.Height)
	fmt.Fprintf(buf, "  CurrentPath:    %q,\n", c.currentPath)
	fmt.Fprintf(buf, "  SelectedId:     %s,\n", f.selectedEntry)
	fmt.Fprintf(buf, "  OffsetIdx:      %d,\n", f.offsetIdx)
	fmt.Fprintf(buf, "  SortColumn:     %s,\n", f.sortColumn)
	fmt.Fprintf(buf, "  SortAscending:  %v,\n", f.sortAscending)
	fmt.Fprintf(buf, "  PendingFiles:   %d,\n", c.pendingFiles)
	fmt.Fprintf(buf, "  DuplicateFiles: %d,\n", c.duplicateFiles)
	fmt.Fprintf(buf, "  AbsentFiles:    %d,\n", c.absentFiles)
	fmt.Fprintf(buf, "  FileTreeLines:  %d,\n", c.fileTreeLines)
	if len(f.entries) > 0 {
		fmt.Fprintf(buf, "  Entries: {\n")
		for _, entry := range f.entries {
			fmt.Fprintf(buf, "    %s:\n", entry)
		}
		fmt.Fprintf(buf, "  }\n")
	}
	if len(c.progressInfos) > 0 {
		fmt.Fprintf(buf, "  Progress: {\n")
		for _, progress := range c.progressInfos {
			fmt.Fprintf(buf, "    {Root: %q, Tab: %q, ETA: %s, Value: %6.2f%%, Speed: %7.3fMb}:\n",
				progress.Root, progress.Tab, progress.TimeRemaining, progress.Value, progress.Speed)
		}
		fmt.Fprintf(buf, "  }\n")
	}
	return buf.String()
}
