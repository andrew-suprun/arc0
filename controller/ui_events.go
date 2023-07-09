package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
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

	if selected.FileKind != w.FileRegular || selected.Status != w.Duplicate {
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

func (c *controller) deleteFile(file *w.File) {
	if file == nil || file.Hash == "" || c.hashStatuses[file.Hash] == w.Pending {
		return
	}
	status := file.Status
	if status != w.Absent {
		return
	}

	if file.FileKind == w.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file)
	}
}

func (c *controller) deleteRegularFile(file *w.File) {
	cmd := m.HandleFiles{Hash: file.Hash}
	for _, root := range c.roots[1:] {
		archive := c.archives[root]
		for _, entry := range archive.files {
			if entry.Hash == file.Hash {
				cmd.Delete = append(cmd.Delete, entry.NewId())
			}
		}
	}
	c.hashStatuses[file.Hash] = w.Pending
	c.archives[c.origin].scanner.Send(cmd)
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: implement
}

func (a *archive) ensureNameAvailable(id m.FileId) *m.RenameFile {
	file := a.fileByNewName(id.FullName())
	if file != nil {
		newName := a.newName(id.FullName())
		file.PendingName = newName
		a.pending[newName] = file
		return &m.RenameFile{FileId: id, NewFullName: newName}
	}
	return nil
}

func (a *archive) newName(name m.FullName) m.FullName {
	parts := strings.Split(name.Name.String(), ".")

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
			if name.Path == entity.Path && newName == entity.Name.String() {
				exists = true
				break
			}
		}
		if !exists {
			return m.FullName{Path: name.Path, Name: m.Name(newName)}
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
