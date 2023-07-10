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
		if folder.selectedId == m.FileId(cmd) && time.Since(c.lastMouseEventTime).Seconds() < 0.5 {
			c.enter()
		} else {
			folder.selectedId = m.FileId(cmd)
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
	folder := c.currentFolder()
	if len(c.entries) > 0 {
		folder.selectedId = c.entries[0].FileId
		folder.offsetIdx = 0
	}
}

func (c *controller) selectLast() {
	folder := c.currentFolder()
	if len(c.entries) > 0 {
		folder.selectedId = c.entries[len(c.entries)-1].FileId
		c.makeSelectedVisible()
	}
}

func (c *controller) enter() {
	selectedId := c.currentFolder().selectedId
	log.Printf("### enter: selectedId: %#v", selectedId)
	var file *w.File
	for i := range c.entries {
		if c.entries[i].FileId == selectedId {
			file = &c.entries[i]
			break
		}
	}

	log.Printf("### enter: file: %q", file)
	if file == nil {
		return
	}
	if file.FileKind == w.FileFolder {
		c.currentPath = m.Path(file.FullName().String())
	} else {
		exec.Command("open", file.String()).Start()
	}
}

func (c *controller) pgUp() {
	c.shiftOffset(-c.feedback.FileTreeLines)
	c.moveSelection(-c.feedback.FileTreeLines)
}

func (c *controller) pgDn() {
	c.shiftOffset(c.feedback.FileTreeLines)
	c.moveSelection(c.feedback.FileTreeLines)
}

func (c *controller) esc() {
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
	selectedId := c.currentFolder().selectedId
	file := c.archives[selectedId.Root].fileByNewName(selectedId.FullName())
	if file != nil {
		exec.Command("open", "-R", file.String()).Start()
	}
}

func (c *controller) moveSelection(lines int) {
	folder := c.currentFolder()

	selectedIdx, _ := m.Find(c.entries, func(entry w.File) bool { return entry.FileId == folder.selectedId })
	selectedIdx += lines
	if selectedIdx < 0 {
		selectedIdx = 0
	}
	if selectedIdx >= len(c.entries) {
		selectedIdx = len(c.entries) - 1
	}
	folder.selectedId = c.entries[selectedIdx].FileId
	c.makeSelectedVisible()
}

func (c *controller) shiftOffset(lines int) {
	folder := c.currentFolder()
	folder.offsetIdx += lines
	if folder.offsetIdx < 0 {
		folder.offsetIdx = 0
	} else if folder.offsetIdx >= len(c.entries) {
		folder.offsetIdx = len(c.entries) - 1
	}
}

func (c *controller) keepSelected() {
	selectedId := c.currentFolder().selectedId
	selectedFile := c.archives[selectedId.Root].fileByNewName(selectedId.FullName())
	c.keepFile(selectedFile)
}

func (c *controller) tab() {
	selectedId := c.currentFolder().selectedId
	selected := c.archives[selectedId.Root].fileByNewName(selectedId.FullName())

	if selected.FileKind != w.FileRegular || c.presence[selected.Hash] != w.Duplicate {
		return
	}
	name := selected.FullName().String()
	hash := selected.Hash
	log.Printf("### tab: name=%q hash=%q", name, hash)
	sameHash := []m.FileId{}
	for _, file := range c.archives[c.origin].files {
		if file.Hash == selected.Hash {
			sameHash = append(sameHash, file.FileId)
		}
	}
	sort.Slice(sameHash, func(i, j int) bool {
		return strings.ToLower(sameHash[i].FullName().String()) < strings.ToLower(sameHash[j].FullName().String())
	})

	idx, _ := m.Find(sameHash, func(id m.FileId) bool { return id == selected.FileId })
	idx++
	if idx == len(sameHash) {
		idx = 0
	}
	id := sameHash[idx]
	c.currentPath = id.Path
	c.currentFolder().selectedId = id

	c.makeSelectedVisible()
}

func (c *controller) deleteSelected() {
	selectedId := c.currentFolder().selectedId
	selected := c.archives[selectedId.Root].fileByNewName(selectedId.FullName())
	c.deleteFile(selected)
}
