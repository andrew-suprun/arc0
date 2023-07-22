package controller

import (
	m "arch/model"
	w "arch/widgets"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (c *controller) mouseTarget(cmd any) {
	folder := c.currentFolder()
	switch cmd := cmd.(type) {
	case m.SelectFile:
		if folder.selectedId == m.Id(cmd) && time.Since(c.lastMouseEventTime).Seconds() < 0.5 {
			c.open()
		} else {
			folder.selectedId = m.Id(cmd)
		}
		c.lastMouseEventTime = time.Now()

	case m.SelectFolder:
		c.currentPath = m.Path(cmd)

	case w.SortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}
	}
}

func (c *controller) selectFirst() {
	if len(c.view.Entries) > 0 {
		folder := c.currentFolder()
		folder.selectedId = c.view.Entries[0].Id
		folder.offsetIdx = 0
	}
}

func (c *controller) selectLast() {
	if len(c.view.Entries) > 0 {
		c.currentFolder().selectedId = c.view.Entries[len(c.view.Entries)-1].Id
		c.makeSelectedVisible()
	}
}

func (c *controller) open() {
	exec.Command("open", c.selectedEntry().Id.String()).Start()
}

func (c *controller) enter() {
	file := c.selectedEntry()
	if file != nil && file.Kind == w.FileFolder {
		c.currentPath = m.Path(file.Name.String())
	}
}

func (c *controller) pgUp() {
	c.shiftOffset(-c.view.FileTreeLines)
	c.moveSelection(-c.view.FileTreeLines)
}

func (c *controller) pgDn() {
	c.shiftOffset(c.view.FileTreeLines)
	c.moveSelection(c.view.FileTreeLines)
}

func (c *controller) exit() {
	if c.currentPath == "" {
		return
	}
	parts := strings.Split(c.currentPath.String(), "/")
	if len(parts) == 1 {
		c.currentPath = ""
	}
	c.currentPath = m.Path(filepath.Join(parts[:len(parts)-1]...))
}

func (c *controller) revealInFinder() {
	exec.Command("open", "-R", c.selectedEntry().Id.String()).Start()
}

func (c *controller) moveSelection(lines int) {
	folder := c.currentFolder()
	id := folder.selectedId
	for idx, entry := range c.view.Entries {
		if entry.Id == id {
			newIdx := idx + lines
			if newIdx >= len(c.view.Entries) {
				newIdx = len(c.view.Entries) - 1
			}
			if newIdx < 0 {
				newIdx = 0
			}
			newId := c.view.Entries[newIdx].Id
			folder.selectedId = newId

		}
	}
	c.makeSelectedVisible()
}

func (c *controller) shiftOffset(lines int) {
	folder := c.currentFolder()
	folder.offsetIdx += lines
	if folder.offsetIdx < 0 {
		folder.offsetIdx = 0
	} else if folder.offsetIdx >= len(c.view.Entries) {
		folder.offsetIdx = len(c.view.Entries) - 1
	}
}

func (c *controller) tab() {
	selected := c.selectedEntry()

	if selected == nil || selected.Kind != w.FileRegular || c.state[selected.Hash] != w.Duplicate {
		return
	}
	sameHash := c.byHash[selected.Hash]

	sort.Slice(sameHash, func(i, j int) bool {
		return strings.ToLower(sameHash[i].Name.String()) < strings.ToLower(sameHash[j].Name.String())
	})

	idx, _ := m.Find(sameHash, func(entry *m.File) bool { return entry.Id == selected.Id })
	for {
		idx = (idx + 1) % len(sameHash)
		if sameHash[idx].Root == c.origin {
			break
		}
	}
	id := sameHash[idx].Id
	c.currentPath = id.Path
	c.currentFolder().selectedId = id

	c.makeSelectedVisible()
}

func (c *controller) keepSelected() {
	selected := c.selectedEntry()
	if selected.Kind == w.FileRegular {
		c.keepFile(&selected.File)
	}
}

func (c *controller) makeSelectedVisible() {
	folder := c.currentFolder()
	selectedIdx := 0
	for idx, entry := range c.view.Entries {
		if entry.Id == folder.selectedId {
			selectedIdx = idx
			break
		}
	}
	offsetIdx := folder.offsetIdx

	if offsetIdx > selectedIdx {
		offsetIdx = selectedIdx
	}
	if offsetIdx < selectedIdx+1-c.view.FileTreeLines {
		offsetIdx = selectedIdx + 1 - c.view.FileTreeLines
	}

	folder.offsetIdx = offsetIdx
}
