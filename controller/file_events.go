package controller

import (
	"arch/model"
	"log"
	"path/filepath"
	"time"
)

func (m *controller) fileMeta(meta model.FileScanned) {
	defer func() {
		m.sort()
	}()

	file := &model.File{
		FileMeta: model.FileMeta(meta),
		Kind:     model.FileRegular,
	}

	m.bySize[meta.Size] = append(m.bySize[meta.Size], file)

	archive := m.archives[meta.Root]
	archive.totalSize += meta.Size
	archive.byINode[meta.INode] = file
	archive.byName[meta.Name] = file

	if m.isOrigin(file.Root) {
		m.addToFolder(file, meta.Size, meta.ModTime)
	}
}

func (m *controller) addToFolder(file *model.File, size uint64, modTime time.Time) {
	parentFolder := m.folders[dir(file.Name)]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &model.File{
				FileMeta: model.FileMeta{
					Name:    dir(file.Name),
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				Kind:   model.FileFolder,
				Status: file.Status,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*model.File{file},
		}
		m.folders[dir(file.Name)] = parentFolder
	} else {
		if file.Kind == model.FileRegular {
			parentFolder.entries = append(parentFolder.entries, file)
			parentFolder.info.Status = parentFolder.info.Status.Merge(file.Status)
		}
		sameFolder := false
		if file.Kind == model.FileFolder {
			for _, entry := range parentFolder.entries {
				if entry.Kind == model.FileFolder && name(file.Name) == name(entry.Name) {
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
	if dir(file.Name) != "" {
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

func (m *controller) makeSelectedVisible() {
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

func (m *controller) fileHashed(fileHash model.FileHashed) {
	archive := m.archives[fileHash.Root]
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
		filesForHash := map[string][]*model.File{}
		for _, file := range filesBySize {
			if file.Hash != hash {
				continue
			}
			filesForHash[file.Root] = append(filesForHash[file.Root], file)
		}
		counts := make([]int, len(m.archives))
		for path, files := range filesForHash {
			counts[m.archiveIdx(path)] = len(files)
		}
		for root := range filesForHash {
			for i := range filesForHash[root] {
				filesForHash[root][i].Counts = counts
			}
		}

		originFiles := filesForHash[m.roots[0]]
		if len(originFiles) == 0 {
			for _, files := range filesForHash {
				for _, file := range files {
					file.Status = model.Conflict
					m.addToFolder(file, file.Size, file.ModTime)
				}
			}
		} else if len(originFiles) == 1 {
			original := originFiles[0]
			m.keepFile(original)
		} else {
			for _, origin := range originFiles {
				origin.Status = model.Conflict
			}
		}
	}
}

func (m *controller) progressEvent(event model.Progress) {
	m.archives[event.Root].progress = event

	if event.ProgressState == model.WalkingFileTreeComplete {
		allWalksComplete := true
		for _, archive := range m.archives {
			if archive.progress.ProgressState != model.WalkingFileTreeComplete {
				allWalksComplete = false
				break
			}
		}
		if allWalksComplete {
			for _, archive := range m.archives {
				archive.scanner.Send(model.HashArchive{})
			}
		}
	}
}

func (m *controller) fileCopied(event model.FileCopied) {
	fromArchive := m.archives[event.FromRoot]
	toArchive := m.archives[event.ToRoot]
	meta := fromArchive.byName[event.Name]
	toArchive.totalCopied += meta.Size
	toArchive.progress.Processed = 0
	log.Printf("### fileCopied %q: %q/%q  copySize=%d  totalCopied=%d",
		event.ToRoot, event.FromRoot, event.Name, toArchive.copySize, toArchive.totalCopied)
	if toArchive.totalCopied == toArchive.copySize {
		toArchive.totalCopied = 0
		toArchive.copySize = 0
		toArchive.progress.ProgressState = model.HashingFileTreeComplete
	}
}

func (m *controller) fileRenamed(meta model.FileRenamed) {
	log.Printf("### fileRenamed: %#v", meta)
}

func (m *controller) fileDeleted(meta model.FileDeleted) {
	log.Printf("### fileDeleted: %#v", meta)
}
