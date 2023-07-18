package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
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
		if c.selectedId() == m.Id(cmd) && time.Since(c.lastMouseEventTime).Seconds() < 0.5 {
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
	if len(c.screen.Entries) > 0 {
		folder := c.currentFolder()
		folder.selectedId = c.screen.Entries[0].Id
		folder.offsetIdx = 0
	}
}

func (c *controller) selectLast() {
	if len(c.screen.Entries) > 0 {
		c.currentFolder().selectedId = c.screen.Entries[len(c.screen.Entries)-1].Id
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
	c.shiftOffset(-c.screen.FileTreeLines)
	c.moveSelection(-c.screen.FileTreeLines)
}

func (c *controller) pgDn() {
	c.shiftOffset(c.screen.FileTreeLines)
	c.moveSelection(c.screen.FileTreeLines)
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
	id := c.selectedId()
	for idx, entry := range c.screen.Entries {
		if entry.Id == id {
			newIdx := idx + lines
			if newIdx >= len(c.screen.Entries) {
				newIdx = len(c.screen.Entries) - 1
			}
			if newIdx < 0 {
				newIdx = 0
			}
			newId := c.screen.Entries[newIdx].Id
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
	} else if folder.offsetIdx >= len(c.screen.Entries) {
		folder.offsetIdx = len(c.screen.Entries) - 1
	}
}

func (c *controller) tab() {
	selected := c.selectedEntry()

	if selected == nil || selected.Kind != w.FileRegular || c.state[selected.Hash] != w.Duplicate {
		return
	}
	sameHash := c.files[selected.Hash]

	sort.Slice(sameHash, func(i, j int) bool {
		return strings.ToLower(sameHash[i].Name.String()) < strings.ToLower(sameHash[j].Name.String())
	})

	for _, entry := range sameHash {
		log.Printf("### entry: %s", entry)
	}

	idx, found := m.Find(sameHash, func(entry *m.File) bool { return entry.Id == selected.Id })
	log.Printf("### idx: %d, found: %v", idx, found)
	for {
		idx = (idx + 1) % len(sameHash)
		if sameHash[idx].Root == c.origin {
			break
		}
	}
	log.Printf("### new idx: %d", idx)
	id := sameHash[idx].Id
	log.Printf("### new id: %q", id)
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
	selectedIdx := c.selectedIdx()
	offsetIdx := c.currentFolder().offsetIdx

	if offsetIdx > selectedIdx {
		offsetIdx = selectedIdx
	}
	if offsetIdx < selectedIdx+1-c.screen.FileTreeLines {
		offsetIdx = selectedIdx + 1 - c.screen.FileTreeLines
	}

	c.currentFolder().offsetIdx = offsetIdx
}
