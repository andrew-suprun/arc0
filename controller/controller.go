package controller

import (
	m "arch/model"
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
	files    map[m.Id]*w.File
	pending  map[m.Id]*w.File
	presence map[m.Hash]w.Presence

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
	progress      m.Progress
	progressState m.ProgressState
	totalSize     uint64
	totalHashed   uint64
	copySize      uint64
	totalCopied   uint64
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
		files:    map[m.Id]*w.File{},
		pending:  map[m.Id]*w.File{},
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

func (c *controller) fileById(id m.Id) *w.File {
	return c.files[id]
}

func (c *controller) fileByNewId(id m.Id) *w.File {
	if result, ok := c.pending[id]; ok {
		return result
	}
	result := c.files[id]
	if result != nil && !result.Pending {
		return result
	}
	return nil
}

func (c *controller) ensureNameAvailable(id m.Id) *m.RenameFile {
	file := c.fileByNewId(id)
	if file != nil {
		newName := c.newName(id)
		file.PendingId = newName
		c.pending[newName] = file
		return &m.RenameFile{Id: id, NewName: newName.Name}
	}
	return nil
}

func (a *controller) newName(id m.Id) m.Id {
	parts := strings.Split(id.Base.String(), ".")

	var part string
	if len(parts) == 1 {
		part = stripIdx(parts[0])
	} else {
		part = stripIdx(parts[len(parts)-2])
	}
	for idx := 1; ; idx++ {
		var newName string
		if len(parts) == 1 {
			newName = fmt.Sprintf("%s [%d]", part, idx)
		} else {
			parts[len(parts)-2] = fmt.Sprintf("%s [%d]", part, idx)
			newName = strings.Join(parts, ".")
		}
		exists := false
		for _, entity := range a.files {
			if id.Path == entity.Path && newName == entity.Base.String() {
				exists = true
				break
			}
		}
		if !exists {
			return m.Id{
				Root: id.Root,
				Name: m.Name{Path: id.Path, Base: m.Base(newName)},
			}
		}
	}
}

type stripIdxState int

const (
	expectCloseBracket stripIdxState = iota
	expectDigit
	expectDigitOrOpenBracket
	expectOpenBracket
	expectSpace
	done
)

func stripIdx(name string) string {
	state := expectCloseBracket
	i := len(name) - 1
	for ; i >= 0; i-- {
		ch := name[i]
		if ch == ']' && state == expectCloseBracket {
			state = expectDigit
		} else if ch >= '0' && ch <= '9' && (state == expectDigit || state == expectDigitOrOpenBracket) {
			state = expectDigitOrOpenBracket
		} else if ch == '[' && state == expectDigitOrOpenBracket {
			state = expectSpace
		} else if ch == ' ' && state == expectSpace {
			break
		} else {
			return name
		}
	}
	return name[:i]
}
