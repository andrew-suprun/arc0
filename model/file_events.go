package model

import (
	"arch/events"
	"log"
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

	m.bySize[meta.Size] = append(m.bySize[meta.Size], file)

	archive := m.archives[m.archiveIdx(meta.ArchivePath)]
	archive.byINode[meta.INode] = file

	if m.archiveIdx(file.ArchivePath) == 0 {
		m.addToFolder(file, meta.Size, meta.ModTime)
	}
}

// TODO: merge file status
func (m *model) addToFolder(file *File, size uint64, modTime time.Time) {
	log.Printf("### addToFolder: path=%q name=%q, status=%v", file.Path, file.Name, file.Status)
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
				Kind:   FileFolder,
				Status: file.Status,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*File{file},
		}
		m.folders[file.Path] = parentFolder
	} else {
		if file.Kind == FileRegular {
			parentFolder.entries = append(parentFolder.entries, file)
			parentFolder.info.Status = parentFolder.info.Status.Merge(file.Status)
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
		parentFolder.info.Status = parentFolder.info.Status.Merge(file.Status)
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

func (m *model) fileHash(hash events.FileHash) {
	archive := m.archives[m.archiveIdx(hash.ArchivePath)]
	file := archive.byINode[hash.INode]
	file.Hash = hash.Hash
	files := m.bySize[file.Size]

	hashes := map[string]struct{}{}
	for _, file := range files {
		hashes[file.Hash] = struct{}{}
	}
	if _, ok := hashes[""]; ok {
		return
	}
	for hash := range hashes {
		log.Printf("### hash %q", hash)
		filesForHash := make([][]*File, len(m.archives))
		for _, file := range files {
			if file.Hash != hash {
				continue
			}
			idx := m.archiveIdx(file.ArchivePath)
			filesForHash[idx] = append(filesForHash[idx], file)
		}
		for idx := range m.archives {
			log.Printf("### archive %q", m.archives[idx].archivePath)
			for _, f := range filesForHash[idx] {
				log.Printf("###      %q %q %q %d %q", f.ArchivePath, f.Path, f.Name, f.Size, f.Hash)
			}
		}
		if len(filesForHash[0]) == 0 {
			for _, files := range filesForHash[1:] {
				for _, file := range files {
					file.Status = CopyOnly
					m.addToFolder(file, file.Size, file.ModTime)
				}
			}
		}
	}
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
