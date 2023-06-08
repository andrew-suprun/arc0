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

	archive := m.archives[m.archiveIdx(meta.ArchivePath)]
	archive.byINode[meta.INode] = file

	if m.archiveIdx(file.ArchivePath) == 0 {
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

func (m *model) fileHash(hash events.FileHash) {
	archive := m.archives[m.archiveIdx(hash.ArchivePath)]
	file := archive.byINode[hash.INode]
	file.Hash = hash.Hash
	filesBySize := m.bySize[file.Size]

	hashes := map[string]struct{}{}
	for _, file := range filesBySize {
		hashes[file.Hash] = struct{}{}
	}
	if _, ok := hashes[""]; ok {
		return
	}
	for hash := range hashes {
		log.Printf("### hash %q", hash)
		filesForHash := make([][]*File, len(m.archives))
		for _, file := range filesBySize {
			if file.Hash != hash {
				continue
			}
			idx := m.archiveIdx(file.ArchivePath)
			filesForHash[idx] = append(filesForHash[idx], file)
		}
		counts := make([]int, len(m.archives))
		for i := range counts {
			counts[i] = len(filesForHash[i])
		}
		for i := range filesForHash {
			for j := range filesForHash[i] {
				filesForHash[i][j].Counts = counts
			}
		}

		for idx := range m.archives {
			log.Printf("### archive %q", m.archives[idx].archivePath)
			for _, f := range filesForHash[idx] {
				log.Printf("###      %q %q %d %q", f.ArchivePath, f.FullName, f.Size, f.Hash)
			}
		}
		if len(filesForHash[0]) == 0 {
			for _, files := range filesForHash[1:] {
				for _, file := range files {
					file.Status = Conflict
					m.addToFolder(file, file.Size, file.ModTime)
				}
			}
		} else if len(filesForHash[0]) == 1 {
			original := filesForHash[0][0]
			for i, copies := range filesForHash[1:] {
				if len(copies) == 0 {
					m.archives[i+1].scanner.Send(files.Copy{From: filesForHash[0][0].FileMeta})
					original.Status = Resolved
				} else {
					moveIdx := 0
					identicalCopyIdx := -1
					for i, copy := range copies {
						if original.FullName == copy.FullName && original.ModTime == copy.ModTime {
							identicalCopyIdx = i
							break
						}
					}
					if identicalCopyIdx >= 0 {
						moveIdx = identicalCopyIdx
					}

					for i, copy := range copies {
						if i != moveIdx {
							m.archives[i+1].scanner.Send(files.Remove{File: copy.FileMeta})
							filesForHash[0][0].Status = Resolved
						}
					}

					if identicalCopyIdx == -1 {
						m.archives[i+1].scanner.Send(files.Move{
							From: filesForHash[0][0].FileMeta,
							To:   copies[0].FileMeta,
						})
					}
				}
			}
		} else {
			for _, origin := range filesForHash[0] {
				origin.Status = Conflict
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
				archive.scanner.Send(files.HashArchive{})
			}
		}
	}
}
