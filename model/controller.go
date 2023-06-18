package model

import (
	"arch/events"
	"arch/files"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (m *model) handleEvent(event any) {
	if event == nil {
		return
	}

	switch event := event.(type) {
	case events.FileMeta:
		m.fileMeta(event)

	case events.FileHash:
		m.fileHash(event)

	case events.Progress:
		m.progressEvent(event)

	case events.ScanError:
		m.Errors = append(m.Errors, event)

	case events.ScreenSize:
		m.screenSize = events.ScreenSize{Width: event.Width, Height: event.Height}

	case events.Enter:
		m.enter()

	case events.Esc:
		if m.currentPath == "" {
			return
		}
		parts := strings.Split(m.currentPath, "/")
		if len(parts) == 1 {
			m.currentPath = ""
		}
		m.currentPath = filepath.Join(parts[:len(parts)-1]...)
		m.sort()

	case events.RevealInFinder:
		folder := m.folders[m.currentPath]
		if folder.selected != nil {
			exec.Command("open", "-R", folder.selected.AbsName()).Start()
		}

	case events.MoveSelection:
		m.moveSelection(event.Lines)
		m.makeSelectedVisible()

	case events.SelectFirst:
		m.selectFirst()
		m.makeSelectedVisible()

	case events.SelectLast:
		m.selectLast()
		m.makeSelectedVisible()

	case events.Scroll:
		m.shiftOffset(event.Lines)

	case events.MouseTarget:
		m.mouseTarget(event.Command)

	case events.PgUp:
		m.shiftOffset(-m.fileTreeLines)
		m.moveSelection(-m.fileTreeLines)

	case events.PgDn:
		m.shiftOffset(m.fileTreeLines)
		m.moveSelection(m.fileTreeLines)

	case events.KeepOne:
		m.keepSelected()

	case events.KeepAll:
		// TODO: Implement, maybe?

	case events.Delete:
		m.deleteSelected()

	case events.Quit:
		m.quit = true

	default:
		log.Panicf("### unhandled event: %#v", event)
	}
}

func (m *model) mouseTarget(cmd any) {
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

func (m *model) selectFirst() {
	folder := m.folders[m.currentPath]
	folder.selected = folder.entries[0]
}

func (m *model) selectLast() {
	folder := m.folders[m.currentPath]
	entries := folder.entries
	folder.selected = entries[len(entries)-1]
}

func (m *model) moveSelection(lines int) {
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

func (m *model) enter() {
	folder := m.folders[m.currentPath]
	selected := folder.selected
	if selected != nil {
		if selected.Kind == FileFolder {
			m.currentPath = selected.FullName
			m.sort()
		} else {
			exec.Command("open", selected.AbsName()).Start()
		}
	}
}

func (m *model) isOrigin(archPath string) bool {
	return archPath == m.archivePaths[0]
}

func (m *model) archiveIdx(archivePath string) int {
	for i := 0; i < len(m.archivePaths); i++ {
		if archivePath == m.archivePaths[i] {
			return i
		}
	}
	log.Panicf("### Invalid archive path: %q", archivePath)
	return -1
}

func (m *model) shiftOffset(lines int) {
	folder := m.folders[m.currentPath]
	nEntries := len(folder.entries)
	folder.lineOffset += lines
	if folder.lineOffset < 0 {
		folder.lineOffset = 0
	} else if folder.lineOffset >= nEntries {
		folder.lineOffset = nEntries - 1
	}
}

func (m *model) keepSelected() {
	m.keepFile(m.folders[m.currentPath].selected)
}

func (m *model) keepFile(file *File) {
	if file == nil || file.Kind != FileRegular {
		return
	}
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}

	for _, archPath := range m.archivePaths {
		archFiles := byArch[archPath]
		if len(archFiles) == 0 {
			archive := m.archives[archPath]
			archive.scanner.Send(files.Copy{Source: file.FileMeta})
			archive.copySize += file.Size
			file.Status = Pending
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
					m.archives[archPath].scanner.Send(files.Move{OldMeta: archFile.FileMeta, NewMeta: file.FileMeta})
					archFile.Status = Pending
				}
			} else {
				m.archives[archPath].scanner.Send(files.Delete{File: archFile.FileMeta})
				archFile.Status = Pending
			}
		}
	}
	m.updateFolderStatus(dir(file.FullName))
	m.sort()
}

func (m *model) updateFolderStatus(path string) {
	currentFolder := m.folders[path]
	status := Identical
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

func (m *model) deleteSelected() {
	m.deleteFile(m.folders[m.currentPath].selected)
}

func (m *model) deleteFile(file *File) {
	if file == nil || file.Status != Conflict {
		return
	}

	if file.Kind == FileFolder {
		m.deleteFolderFile(file)
	} else {
		m.deleteRegularFile(file)
	}
	m.updateFolderStatus(dir(file.FullName))
	m.sort()
}

func (m *model) deleteRegularFile(file *File) {
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}
	if len(byArch[m.archivePaths[0]]) > 0 {
		return
	}

	for _, file := range filesForHash {
		m.archives[file.ArchivePath].scanner.Send(files.Delete{File: file.FileMeta})
		file.Status = Pending
	}
}

func (m *model) deleteFolderFile(file *File) {
	folder := m.folders[file.FullName]
	for _, entry := range folder.entries {
		m.deleteFile(entry)
	}
}
