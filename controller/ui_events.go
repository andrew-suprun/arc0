package controller

import (
	"arch/model"
	"log"
	"os/exec"
	"time"
)

func (m *controller) mouseTarget(cmd any) {
	folder := m.folders[m.currentPath]
	switch cmd := cmd.(type) {
	case selectFile:
		if folder.selected == cmd && time.Since(m.lastMouseEventTime).Seconds() < 0.5 {
			m.enter()
		} else {
			folder.selected = cmd
		}
		m.lastMouseEventTime = time.Now()

	case selectFolder:
		m.currentPath = cmd.Name

	case sortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}
	}
}

func (m *controller) selectFirst() {
	folder := m.folders[m.currentPath]
	folder.selected = folder.entries[0]
}

func (m *controller) selectLast() {
	folder := m.folders[m.currentPath]
	entries := folder.entries
	folder.selected = entries[len(entries)-1]
}

func (m *controller) moveSelection(lines int) {
	folder := m.folders[m.currentPath]
	selected := folder.selected
	if selected == nil {
		if lines > 0 {
			m.selectFirst()
		} else if lines < 0 {
			m.selectLast()
		}
	}
	entries := folder.entries
	idxSelected := 0
	foundSelected := false

	for i := 0; i < len(entries); i++ {
		if entries[i] == selected {
			idxSelected = i
			foundSelected = true
			break
		}
	}
	if foundSelected {
		idxSelected += lines
		if idxSelected < 0 {
			idxSelected = 0
		} else if idxSelected >= len(entries) {
			idxSelected = len(entries) - 1
		}
		folder.selected = entries[idxSelected]
	}
}

func (m *controller) enter() {
	folder := m.folders[m.currentPath]
	selected := folder.selected
	if selected != nil {
		if selected.Kind == model.FileFolder {
			m.currentPath = selected.Name
		} else {
			exec.Command("open", selected.AbsName()).Start()
		}
	}
}

func (m *controller) archiveIdx(root string) int {
	for i := 0; i < len(m.roots); i++ {
		if root == m.roots[i] {
			return i
		}
	}
	log.Panicf("### Invalid archive path: %q", root)
	return -1
}

func (m *controller) shiftOffset(lines int) {
	folder := m.folders[m.currentPath]
	nEntries := len(folder.entries)
	folder.lineOffset += lines
	if folder.lineOffset < 0 {
		folder.lineOffset = 0
	} else if folder.lineOffset >= nEntries {
		folder.lineOffset = nEntries - 1
	}
}

func (m *controller) keepSelected() {
	m.keepFile(m.folders[m.currentPath].selected)
}

func (m *controller) keepFile(file *model.File) {
	if file == nil || file.Kind != model.FileRegular {
		return
	}
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*model.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}

	for _, root := range m.roots {
		archFiles := byArch[root]
		if len(archFiles) == 0 {
			archive := m.archives[root]
			archive.scanner.Send(model.CopyFile{
				Root: file.Root,
				Name: file.Name,
			})
			archive.copySize += file.Size
			file.Status = model.Pending
			log.Printf("### keepFile %q: root=%q  name=%q  copySize=%d", root, file.Root, file.Name, archive.copySize)
			continue
		}
		keepIdx := 0
		for i, archFile := range archFiles {
			if archFile == file || archFile.Name == file.Name {
				keepIdx = i
				break
			}
		}
		for i, archFile := range archFiles {
			if i == keepIdx {
				if file.Name != archFile.Name {
					m.archives[root].scanner.Send(model.RenameFile{OldName: archFile.FileMeta.Name, NewName: file.FileMeta.Name})
					archFile.Status = model.Pending
				}
			} else {
				m.archives[root].scanner.Send(model.DeleteFile{Name: archFile.FileMeta.Name})
				archFile.Status = model.Pending
			}
		}
	}
	m.updateFolderStatus(dir(file.Name))
}

func (m *controller) updateFolderStatus(path string) {
	currentFolder := m.folders[path]
	status := model.Identical
	for _, entry := range currentFolder.entries {
		status = status.Merge(entry.Status)
	}
	if currentFolder.info.Status == status {
		return
	}
	currentFolder.info.Status = status
	if path == "" {
		return
	}
	m.updateFolderStatus(dir(path))
}

func (m *controller) deleteSelected() {
	m.deleteFile(m.folders[m.currentPath].selected)
}

func (m *controller) deleteFile(file *model.File) {
	if file == nil || file.Status != model.Conflict {
		return
	}

	if file.Kind == model.FileFolder {
		m.deleteFolderFile(file)
	} else {
		m.deleteRegularFile(file)
	}
	m.updateFolderStatus(dir(file.Name))
}

func (m *controller) deleteRegularFile(file *model.File) {
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*model.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}
	if len(byArch[m.roots[0]]) > 0 {
		return
	}

	for _, file := range filesForHash {
		m.archives[file.Root].scanner.Send(model.DeleteFile{Name: file.FileMeta.Name})
		file.Status = model.Pending
	}
}

func (m *controller) deleteFolderFile(file *model.File) {
	folder := m.folders[file.Name]
	for _, entry := range folder.entries {
		m.deleteFile(entry)
	}
}
