package model

import (
	"arch/events"
	"path/filepath"
	"time"
)

func (m *model) fileMeta(meta events.FileMeta) {
	defer func() {
		m.analyze()
		m.sort()
	}()

	file := &File{
		FileMeta: meta,
		Kind:     FileRegular,
	}

	archive := m.archives[m.archiveIdx(meta.ArchivePath)]
	archive.byIno[meta.Ino] = file

	if m.archiveIdx(file.ArchivePath) == 0 {
		m.addToFolder(file, meta.Size, meta.ModTime)
	}
}

func (m *model) addToFolder(file *File, size uint64, modTime time.Time) {
	parentFolder := m.folders[file.Path]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &File{
				FileMeta: events.FileMeta{
					Path:    dir(file.Path),
					Name:    filepath.Base(file.Path),
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				Kind: FileFolder,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*File{file},
		}
		m.folders[file.Path] = parentFolder
	} else {
		if file.Kind == FileRegular {
			parentFolder.entries = append(parentFolder.entries, file)
		}
		sameFolder := false
		if file.Kind == FileFolder {
			for _, entry := range parentFolder.entries {
				if file.Name == entry.Name && entry.Kind == FileFolder {
					sameFolder = true
					break
				}
			}
			if !sameFolder {
				parentFolder.entries = append(parentFolder.entries, file)
			}
		}
		parentFolder.info.Size += size
		if parentFolder.info.ModTime.Before(modTime) {
			parentFolder.info.ModTime = modTime
		}
	}
	if file.Path != "" {
		m.addToFolder(parentFolder.info, size, modTime)
	}
}

func dir(path string) string {
	path = filepath.Dir(path)
	if path == "." {
		return ""
	}
	return path
}

func (m *model) makeSelectedVisible() {
	folder := m.currentFolder()
	if folder.selected == nil {
		return
	}

	idx := -1
	for i := range folder.entries {
		if folder.selected == folder.entries[i] {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if folder.lineOffset > idx {
			folder.lineOffset = idx
		}
		if folder.lineOffset < idx+1-m.fileTreeLines {
			folder.lineOffset = idx + 1 - m.fileTreeLines
		}
	}
}

func (m *model) fileHash(hash events.FileHash) {
	// TODO: implement
}

func (m *model) scanProgressEvent(event events.ScanProgress) {
	m.archives[m.archiveIdx(event.ArchivePath)].scanState = event

	if event.ScanState == events.WalkFileTreeComplete {
		allWalksComplete := true
		for _, archive := range m.archives {
			if archive.scanState.ScanState != events.WalkFileTreeComplete {
				allWalksComplete = false
				break
			}
		}
		if allWalksComplete {
			for _, archive := range m.archives {
				archive.scanner.HashArchive()
			}
		}
	}
}
