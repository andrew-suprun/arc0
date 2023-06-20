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
		m.currentPath = cmd.FullName

	case sortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}

		m.sort()
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
			m.currentPath = selected.FullName
			m.sort()
		} else {
			exec.Command("open", selected.AbsName()).Start()
		}
	}
}

func (m *controller) isOrigin(archPath string) bool {
	return archPath == m.archivePaths[0]
}

func (m *controller) archiveIdx(archivePath string) int {
	for i := 0; i < len(m.archivePaths); i++ {
		if archivePath == m.archivePaths[i] {
			return i
		}
	}
	log.Panicf("### Invalid archive path: %q", archivePath)
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
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}

	for _, archPath := range m.archivePaths {
		archFiles := byArch[archPath]
		if len(archFiles) == 0 {
			archive := m.archives[archPath]
			archive.scanner.Send(model.CopyFile(file.FileMeta))
			archive.copySize += file.Size
			file.Status = model.Pending
			continue
		}
		keepIdx := 0
		for i, archFile := range archFiles {
			if archFile == file || archFile.FullName == file.FullName {
				keepIdx = i
				break
			}
		}
		for i, archFile := range archFiles {
			if i == keepIdx {
				if file.FullName != archFile.FullName {
					m.archives[archPath].scanner.Send(model.RenameFile{OldMeta: archFile.FileMeta, NewMeta: file.FileMeta})
					archFile.Status = model.Pending
				}
			} else {
				m.archives[archPath].scanner.Send(model.DeleteFile(archFile.FileMeta))
				archFile.Status = model.Pending
			}
		}
	}
	m.updateFolderStatus(dir(file.FullName))
	m.sort()
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
	m.updateFolderStatus(dir(file.FullName))
	m.sort()
}

func (m *controller) deleteRegularFile(file *model.File) {
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*model.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}
	if len(byArch[m.archivePaths[0]]) > 0 {
		return
	}

	for _, file := range filesForHash {
		m.archives[file.ArchivePath].scanner.Send(model.DeleteFile(file.FileMeta))
		file.Status = model.Pending
	}
}

func (m *controller) deleteFolderFile(file *model.File) {
	folder := m.folders[file.FullName]
	for _, entry := range folder.entries {
		m.deleteFile(entry)
	}
}
