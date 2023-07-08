package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

type screenBuilder struct {
	*controller
	copyNameHash map[m.Name]m.Hash
	absentHashes map[m.Hash]struct{}
	originHashed bool
}

func (c *controller) buildScreen() *w.Screen {
	builder := &screenBuilder{
		controller:   c,
		copyNameHash: map[m.Name]m.Hash{},
		absentHashes: map[m.Hash]struct{}{},
	}
	builder.buildEntries()

	folder := c.folders[c.currentPath]
	screen := &w.Screen{
		CurrentPath:   c.currentPath,
		Progress:      c.progress(),
		SelectedId:    folder.selectedId,
		OffsetIdx:     folder.offsetIdx,
		SortColumn:    folder.sortColumn,
		SortAscending: folder.sortAscending,
	}

	screen.Entries = make([]w.File, len(c.entries))
	copy(screen.Entries, c.entries)
	return screen
}

func (b *screenBuilder) buildEntries() {
	b.controller.entries = b.controller.entries[:0]

	b.handleOrigin(b.controller.archives[b.controller.origin])

	for _, root := range b.controller.roots[1:] {
		b.handleCopy(b.controller.archives[root])
	}

	b.controller.sort()
}

func (b *screenBuilder) handleOrigin(archive *archive) {
	for _, file := range archive.files {
		if file.Path == b.controller.currentPath {
			b.controller.entries = append(b.controller.entries, w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
				Hash:     file.Hash,
				Status:   file.Status,
			})
		} else if strings.HasPrefix(file.Path.String(), b.controller.currentPath.String()) {
			relPath := file.Path
			if len(b.controller.currentPath) > 0 {
				relPath = file.Path[len(b.controller.currentPath)+1:]
			}
			name := m.Name(strings.SplitN(relPath.String(), "/", 2)[0])

			i, found := m.Find(b.controller.entries, func(entry w.File) bool { return name == entry.Name })
			if found {
				b.controller.entries[i].Size += file.Size
				if b.controller.entries[i].ModTime.Before(file.ModTime) {
					b.controller.entries[i].ModTime = file.ModTime
				}
				if file.Status == w.Duplicate {
					b.controller.entries[i].Status = w.Duplicate
				}
			} else {
				entry := w.File{
					FileMeta: m.FileMeta{
						FileId: m.FileId{
							Root: file.Root,
							Path: b.controller.currentPath,
							Name: name,
						},
						Size:    file.Size,
						ModTime: file.ModTime,
					},
					FileKind: w.FileFolder,
				}
				if file.Status == w.Duplicate {
					entry.Status = w.Duplicate
				}
				b.controller.entries = append(b.controller.entries, entry)
			}
		}
	}
}

func (b screenBuilder) handleCopy(archive *archive) {
	if !b.originHashed {
		return
	}

	for _, file := range archive.files {
		if file.Status != w.Absent {
			continue
		}
		if hash, ok := b.copyNameHash[file.Name]; ok && file.Hash == hash {
			continue
		}
		if file.Path == b.controller.currentPath {
			entry := w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
				Hash:     file.Hash,
				Status:   w.Absent,
			}

			b.controller.entries = append(b.controller.entries, entry)
			b.copyNameHash[entry.Name] = entry.Hash
		} else if strings.HasPrefix(file.Path.String(), b.controller.currentPath.String()) {
			relPath := file.Path
			if len(b.controller.currentPath) > 0 {
				relPath = file.Path[len(b.controller.currentPath)+1:]
			}
			name := m.Name(strings.SplitN(relPath.String(), "/", 2)[0])

			_, found := m.Find(b.controller.entries, func(entry w.File) bool { return name == entry.Name })
			if found {
				continue
			}
			entry := w.File{
				FileMeta: m.FileMeta{
					FileId: m.FileId{
						Root: file.Root,
						Path: b.controller.currentPath,
						Name: name,
					},
				},
				FileKind: w.FileFolder,
				Status:   w.Absent,
			}
			b.controller.entries = append(b.controller.entries, entry)

		}
	}
}

func (c *controller) progress() []w.ProgressInfo {
	infos := []w.ProgressInfo{}
	var tab string
	var value float64
	for _, root := range c.roots {
		archive := c.archives[root]
		if archive.progressState == m.FileTreeScanned {
			tab = " Hashing"
			value = float64(archive.totalHashed+archive.progress.HandledSize) / float64(archive.totalSize)
			infos = append(infos, w.ProgressInfo{Root: root, Tab: tab, Value: value})
		} else if archive.progressState == m.FileTreeHashed && archive.copySize != 0 {
			tab = " Copying"
			value = float64(archive.totalCopied+archive.progress.HandledSize) / float64(archive.copySize)
			infos = append(infos, w.ProgressInfo{Root: root, Tab: tab, Value: value})
		}
	}
	return infos
}
