package model

import (
	"arch/files"
)

func (m *model) keepSelected() {
	m.keepFile(m.folders[m.currentPath].selected)
}

func (m *model) keepFile(file *File) {
	if file == nil {
		return
	}
	if file.Kind == FileRegular {
		m.keepRegularFile(file)
	} else {
		m.keepFolderFile(file)
	}
	m.updateFolderStatus(dir(file.FullName))
	m.sort()
}

func (m *model) keepRegularFile(file *File) {
	filesForHash := m.byHash[file.Hash]
	byArch := map[string][]*File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.ArchivePath] = append(byArch[fileForHash.ArchivePath], fileForHash)
	}

	for _, archPath := range m.archivePaths {
		archFiles := byArch[archPath]
		if len(archFiles) == 0 {
			m.archives[archPath].scanner.Send(files.Copy{Source: file.FileMeta})
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
}

func (m *model) keepFolderFile(file *File) {
	folder := m.folders[file.FullName]
	for _, entry := range folder.entries {
		m.keepFile(entry)
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
