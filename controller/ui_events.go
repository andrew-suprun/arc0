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
	folder := c.folders[c.currentPath]
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
	folder := c.folders[c.currentPath]
	folder.selectedId = c.entries[0].FileId
	folder.offsetIdx = 0
}

func (c *controller) selectLast() {
	folder := c.folders[c.currentPath]
	folder.selectedId = c.entries[len(c.entries)-1].FileId
	c.makeSelectedVisible()
}

func (c *controller) enter() {
	selectedId := c.folders[c.currentPath].selectedId
	file := c.archives[selectedId.Root].files[selectedId.FullName()]
	if file == nil {
		return
	}
	if file.FileKind == w.FileFolder {
		c.currentPath = m.Path(file.FullName().String())
	} else {
		exec.Command("open", file.AbsName()).Start()
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
	selectedId := c.folders[c.currentPath].selectedId
	file := c.archives[selectedId.Root].files[selectedId.FullName()]
	if file != nil {
		exec.Command("open", "-R", file.AbsName()).Start()
	}
}

func (c *controller) moveSelection(lines int) {
	folder := c.folders[c.currentPath]

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
	folder := c.folders[c.currentPath]
	folder.offsetIdx += lines
	if folder.offsetIdx < 0 {
		folder.offsetIdx = 0
	} else if folder.offsetIdx >= len(c.entries) {
		folder.offsetIdx = len(c.entries) - 1
	}
}

func (c *controller) keepSelected() {
	selectedId := c.folders[c.currentPath].selectedId
	selectedFile := c.archives[selectedId.Root].files[selectedId.FullName()]
	c.keepFile(selectedFile)
}

func (c *controller) keepFile(file *w.File) {
	if file == nil || file.FileKind != w.FileRegular || file.Hash == "" || file.Status == w.Pending {
		return
	}
	log.Printf("### keep %v", file)

	keepFiles := map[m.Root]*w.File{}
	for root, archive := range c.archives {
		for _, entry := range archive.files {
			if entry.Hash == file.Hash {
				if prevFile, ok := keepFiles[root]; ok {
					if entry.FullName() == file.FullName() {
						keepFiles[root] = entry
					} else if entry.Path == file.Path && entry.Path != prevFile.Path {
						keepFiles[root] = entry
					} else if entry.Name == file.Name && entry.Name != prevFile.Name {
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
			if entry.Hash == file.Hash {
				keepFile := keepFiles[root]
				if entry == keepFile {
					if keepFile.FullName() != file.FullName() {
						archive.scanner.Send(m.RenameFile{FileId: keepFile.FileId, NewFullName: file.FullName()})
						log.Printf("+++ rename.1 %v", m.RenameFile{FileId: keepFile.FileId, NewFullName: file.FullName()})
						keepFile.Status = w.Pending
					}
				} else {
					archive.scanner.Send(m.DeleteFile(entry.FileId))
					log.Printf("+++ delete %v", m.DeleteFile(entry.FileId))
					entry.Status = w.Pending
				}
			}
		}
	}

	nameByHash := map[m.Hash]m.Name{}
	for _, entry := range c.entries {
		if entry.Name == file.Name && entry.Root == c.origin {
			nameByHash[entry.Hash] = entry.Name
		}
	}
	if len(nameByHash) == 0 {
		nameByHash[file.Hash] = file.Name
	}

	for _, entry := range c.entries {
		if entry.Name == file.Name {
			if _, ok := nameByHash[entry.Hash]; !ok {
				nameByHash[entry.Hash] = c.newName(entry.Name)
			}
			newName := nameByHash[entry.Hash]
			if newName != entry.Name {
				c.archives[entry.Root].scanner.Send(m.RenameFile{
					FileId: entry.FileId,
					NewFullName: m.FullName{
						Path: entry.Path,
						Name: newName,
					},
				})
				log.Printf("+++ rename.2 %v", m.RenameFile{
					FileId: entry.FileId,
					NewFullName: m.FullName{
						Path: entry.Path,
						Name: newName,
					},
				})
				entry.Status = w.Pending
			}
		}
	}

	for root, archive := range c.archives {
		if _, ok := keepFiles[root]; !ok {
			archive.scanner.Send(m.CopyFile{
				From: file.FileId,
				To:   root,
			})
			log.Printf("+++ copy %v", m.CopyFile{
				From: file.FileId,
				To:   root,
			})
			file.Status = w.Pending
			archive.totalSize += file.Size
		}
	}
}

func (c *controller) tab() {
	selectedId := c.folders[c.currentPath].selectedId
	selected := c.archives[selectedId.Root].files[selectedId.FullName()]

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
	c.folders[c.currentPath].selectedId = id

	c.makeSelectedVisible()
}

func (c *controller) deleteSelected() {
	selectedId := c.folders[c.currentPath].selectedId
	selected := c.archives[selectedId.Root].files[selectedId.FullName()]
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
	c.archives[file.Root].scanner.Send(m.DeleteFile(file.FileId))
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: need it?
}

func (c *controller) newName(name m.Name) m.Name {
	parts := strings.Split(name.String(), ".")

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
		for _, entity := range c.entries {
			if newName == entity.Name.String() {
				exists = true
				break
			}
		}
		if !exists {
			return m.Name(newName)
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
