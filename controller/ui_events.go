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

func (c *controller) keepFile(file *w.File) {
	if file == nil || file.FileKind != w.FileRegular || file.Hash == "" || file.Status == w.Pending {
		return
	}
	fileId := file.NewId()
	fileName := fileId.FullName()
	log.Printf("### keep %#v", fileId)
	pending := false

	keepFiles := map[m.Root]*w.File{}
	for root, archive := range c.archives {
		for _, entry := range archive.files {
			if entry.Hash == file.Hash {
				name := entry.NewName()
				if prevFile, ok := keepFiles[root]; ok {
					if name == fileId.FullName() {
						keepFiles[root] = entry
					} else if name.Path == fileId.Path && name.Path != prevFile.Path {
						keepFiles[root] = entry
					} else if name.Name == fileId.Name && name.Name != prevFile.Name {
						keepFiles[root] = entry
					}
				} else {
					keepFiles[root] = entry
				}
			}
		}
	}

	for root, archive := range c.archives {
		for _, entry := range archive.files {
			if entry == file {
				continue
			}
			if entry.Hash == file.Hash {
				keepFile := keepFiles[root]
				if entry == keepFile {
					if fileName != keepFile.FullName() {
						newId := m.FileId{Root: keepFile.Root, Path: fileId.Path, Name: fileId.Name}
						archive.ensureNameAvailable(newId)
						archive.scanner.Send(m.RenameFile{FileId: keepFile.NewId(), NewFullName: fileName})
						log.Printf("+++ rename.1 %#v", m.RenameFile{FileId: keepFile.NewId(), NewFullName: fileName})
						pending = true
						keepFile.Status = w.Pending
						keepFile.PendingName = fileName
						archive.pending[fileName] = keepFile
					}
				} else {
					id := entry.NewId()
					name := id.FullName()
					archive.scanner.Send(m.DeleteFile(id))
					log.Printf("+++ delete.1 %#v", m.DeleteFile(id))

					delete(archive.files, name)
					delete(archive.pending, name)
					// pending = true
					// entry.Status = w.Pending
				}
			}
		}
	}

	for root, archive := range c.archives {
		if root == fileId.Root {
			continue
		}
		if _, ok := keepFiles[root]; !ok {
			newId := m.FileId{Root: root, Path: fileName.Path, Name: fileName.Name}
			archive.ensureNameAvailable(newId)
			archive.scanner.Send(m.CopyFile{From: fileId, To: newId.Root})
			log.Printf("+++ copy %#v", m.CopyFile{From: fileId, To: newId.Root})
			pending = true
			file.Status = w.Pending
			archive.copySize += file.Size
		}
	}
	if pending {
		c.hashStatuses[file.Hash] = w.Pending
	}
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
	if file == nil {
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
	archive := c.archives[file.Root]

	id := file.NewId()
	name := id.FullName()
	archive.scanner.Send(m.DeleteFile(id))
	log.Printf("+++ delete.2 %#v", m.DeleteFile(id))

	delete(archive.files, name)
	delete(archive.pending, name)
	// file.Status = w.Pending
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: need it?
}

func (a *archive) ensureNameAvailable(id m.FileId) {
	file := a.fileByNewName(id.FullName())
	if file != nil {
		newName := a.newName(id.FullName())
		a.scanner.Send(m.RenameFile{FileId: id, NewFullName: newName})
		log.Printf("+++ rename.2 %#v", m.RenameFile{FileId: id, NewFullName: newName})
		file.Status = w.Pending
		file.PendingName = newName
		a.pending[newName] = file
	}
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
