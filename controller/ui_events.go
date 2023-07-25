package controller

import (
	m "arch/model"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

func (c *controller) mouseTarget(cmd any) {
	folder := c.currentFolder()
	selected := folder.selectedEntry
	switch cmd := cmd.(type) {
	case m.SelectFile:
		if selected.Id == m.Id(cmd) && time.Since(c.lastMouseEventTime).Seconds() < 0.5 {
			c.open()
		} else {
			folder.selectedEntry = folder.entries[cmd.Base]
		}
		c.lastMouseEventTime = time.Now()

	case m.SelectFolder:
		c.currentPath = m.Path(cmd)

	case m.SortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}
	}
}

func (c *controller) selectFirst() {
	folder := c.currentFolder()
	if len(folder.entries) == 0 {
		return
	}
	sorted := folder.sort()
	folder.selectedEntry = sorted[0]
	folder.offsetIdx = 0
}

func (c *controller) selectLast() {
	folder := c.currentFolder()
	if len(folder.entries) == 0 {
		return
	}
	sorted := folder.sort()
	folder.selectedEntry = sorted[len(folder.entries)-1]
	c.makeSelectedVisible()
}

func (c *controller) open() {
	exec.Command("open", c.currentFolder().selectedEntry.Id.String()).Start()
}

func (c *controller) enter() {
	file := c.currentFolder().selectedEntry
	if file != nil && file.Kind == m.FileFolder {
		c.currentPath = m.Path(file.Name.String())
	}
}

func (c *controller) pgUp() {
	c.shiftOffset(-c.fileTreeLines)
	c.moveSelection(-c.fileTreeLines)
}

func (c *controller) pgDn() {
	c.shiftOffset(c.fileTreeLines)
	c.moveSelection(c.fileTreeLines)
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
	exec.Command("open", "-R", c.currentFolder().selectedEntry.Id.String()).Start()
}

func (c *controller) moveSelection(lines int) {
	folder := c.currentFolder()
	if len(folder.entries) == 0 {
		return
	}
	id := folder.selectedEntry.Id
	sorted := folder.sort()
	for idx, entry := range sorted {
		if entry.Id == id {
			newIdx := idx + lines
			if newIdx >= len(folder.entries) {
				newIdx = len(folder.entries) - 1
			}
			if newIdx < 0 {
				newIdx = 0
			}
			folder.selectedEntry = sorted[newIdx]

		}
	}
	c.makeSelectedVisible()
}

func (c *controller) shiftOffset(lines int) {
	folder := c.currentFolder()
	folder.offsetIdx += lines
	if folder.offsetIdx < 0 {
		folder.offsetIdx = 0
	} else if folder.offsetIdx >= len(folder.entries) {
		folder.offsetIdx = len(folder.entries) - 1
	}
}

func (c *controller) tab() {
	selected := c.currentFolder().selectedEntry

	if selected == nil || selected.Kind != m.FileRegular || selected.State != m.Duplicate {
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
	c.currentPath = sameHash[idx].Path
	c.currentFolder().selectedEntry = sameHash[idx]
	c.makeSelectedVisible()
}

func (c *controller) keepSelected() {
	selected := c.currentFolder().selectedEntry
	if selected.Kind == m.FileRegular {
		c.keepFile(selected)
	}
}

func (c *controller) makeSelectedVisible() {
	folder := c.currentFolder()
	selectedIdx := 0
	sorted := folder.sort()
	for idx, entry := range sorted {
		if entry == folder.selectedEntry {
			selectedIdx = idx
			break
		}
	}
	offsetIdx := folder.offsetIdx

	if offsetIdx > selectedIdx {
		offsetIdx = selectedIdx
	}
	if offsetIdx < selectedIdx+1-c.fileTreeLines {
		offsetIdx = selectedIdx + 1 - c.fileTreeLines
	}

	folder.offsetIdx = offsetIdx
}
