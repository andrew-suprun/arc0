package model

import (
	"arch/events"
	"arch/files"
	"log"
	"path/filepath"
	"time"
)

func (m *model) fileMeta(meta events.FileMeta) {
	defer func() {
		m.sort()
	}()

	file := &File{
		FileMeta: meta,
		Kind:     FileRegular,
	}

	m.bySize[meta.Size] = append(m.bySize[meta.Size], file)

	archive := m.archives[meta.ArchivePath]
	archive.totalSize += meta.Size
	archive.byINode[meta.INode] = file

	if m.isOrigin(file.ArchivePath) {
		m.addToFolder(file, meta.Size, meta.ModTime)
	}
}

func (m *model) addToFolder(file *File, size uint64, modTime time.Time) {
	parentFolder := m.folders[dir(file.FullName)]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &File{
				FileMeta: events.FileMeta{
					FullName: dir(file.FullName),
					Size:     file.Size,
					ModTime:  file.ModTime,
				},
				Kind:   FileFolder,
				Status: file.Status,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*File{file},
		}
		m.folders[dir(file.FullName)] = parentFolder
	} else {
		if file.Kind == FileRegular {
			parentFolder.entries = append(parentFolder.entries, file)
			parentFolder.info.Status = parentFolder.info.Status.Merge(file.Status)
		}
		sameFolder := false
		if file.Kind == FileFolder {
			for _, entry := range parentFolder.entries {
				if entry.Kind == FileFolder && name(file.FullName) == name(entry.FullName) {
					sameFolder = true
					break
				}
			}
			if !sameFolder {
				parentFolder.entries = append(parentFolder.entries, file)
			}
		}
		parentFolder.info.Status = parentFolder.info.Status.Merge(file.Status)
		parentFolder.info.Size += size
		if parentFolder.info.ModTime.Before(modTime) {
			parentFolder.info.ModTime = modTime
		}
	}
	if dir(file.FullName) != "" {
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

func name(path string) string {
	return filepath.Base(path)
}

func (m *model) makeSelectedVisible() {
	folder := m.folders[m.currentPath]
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

func (m *model) fileHash(fileHash events.FileHash) {
	archive := m.archives[fileHash.ArchivePath]
	file := archive.byINode[fileHash.INode]
	file.Hash = fileHash.Hash
	filesBySize := m.bySize[file.Size]
	m.byHash[fileHash.Hash] = append(m.byHash[fileHash.Hash], file)

	hashes := map[string]struct{}{}
	for _, file := range filesBySize {
		hashes[file.Hash] = struct{}{}
	}
	if _, ok := hashes[""]; ok {
		return
	}
	for hash := range hashes {
		filesForHash := map[string][]*File{}
		for _, file := range filesBySize {
			if file.Hash != hash {
				continue
			}
			filesForHash[file.ArchivePath] = append(filesForHash[file.ArchivePath], file)
		}
		counts := make([]int, len(m.archives))
		for path, files := range filesForHash {
			counts[m.archiveIdx(path)] = len(files)
		}
		for archPath := range filesForHash {
			for i := range filesForHash[archPath] {
				filesForHash[archPath][i].Counts = counts
			}
		}

		originFiles := filesForHash[m.archivePaths[0]]
		if len(originFiles) == 0 {
			for _, files := range filesForHash {
				for _, file := range files {
					file.Status = Conflict
					m.addToFolder(file, file.Size, file.ModTime)
				}
			}
		} else if len(originFiles) == 1 {
			original := originFiles[0]
			m.keepFile(original)
		} else {
			for _, origin := range originFiles {
				origin.Status = Conflict
			}
		}
	}
}

func (m *model) progressEvent(event events.Progress) {
	log.Printf("### progress: event=%#v", event)
	m.archives[event.ArchivePath].progress = event

	if event.ProgressState == events.WalkFileTreeComplete {
		allWalksComplete := true
		for _, archive := range m.archives {
			if archive.progress.ProgressState != events.WalkFileTreeComplete {
				allWalksComplete = false
				break
			}
		}
		if allWalksComplete {
			for _, archive := range m.archives {
				archive.scanner.Send(files.HashArchive{})
			}
		}
	}
}
