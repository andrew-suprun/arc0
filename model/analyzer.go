package model

import (
	"arch/files"
)

func (m *model) keepOneFile(selected *File) {
	if selected.Status != Conflict {
		return
	}
	if selected.Kind == FileRegular {
		m.keepOneRegularFile(selected)
	} else {
		m.keepOneFolderFile(selected)
	}
	for _, archFile := range m.byHash[selected.Hash] {
		archFile.Status = Pending
	}
	m.updateFolderStatus(dir(selected.FullName))
	m.sort()
}

func (m *model) keepOneRegularFile(selected *File) {
	filesForHash := m.byHash[selected.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}

	for _, archPath := range m.archivePaths {
		archFiles := byArch[archPath]
		if len(archFiles) == 0 {
			m.archives[archPath].scanner.Send(files.Copy{Source: selected.FileMeta})
			continue
		}
		keepIdx := 0
		for i, archFile := range archFiles {
			if archFile == selected || archFile.FullName == selected.FullName {
				keepIdx = i
				break
			}
		}
		for i, archFile := range archFiles {
			if i == keepIdx {
				if selected.FullName != archFile.FullName {
					m.archives[archPath].scanner.Send(files.Move{OldMeta: archFile.FileMeta, NewMeta: selected.FileMeta})
				}
			} else {
				m.archives[archPath].scanner.Send(files.Delete{File: archFile.FileMeta})
			}
		}
	}
}

func (m *model) keepOneFolderFile(selected *File) {
	folder := m.folders[selected.FullName]
	for _, entry := range folder.entries {
		m.keepOneFile(entry)
	}
}

func (m *model) updateFolderStatus(path string) {
	currentFolder := m.folders[path]
	status := Identical
	for _, entry := range currentFolder.entries {
		status = status.Merge(entry.Status)
	}
	currentFolder.info.Status = status
	if path == "" {
		return
	}
	m.updateFolderStatus(dir(path))
}
