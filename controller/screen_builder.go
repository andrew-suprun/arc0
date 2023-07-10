package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
	"strings"
)

type screenBuilder struct {
	copyNameHash map[m.Name]m.Hash
	originHashed bool
}

func (c *controller) buildScreen() *w.Screen {
	builder := &screenBuilder{
		copyNameHash: map[m.Name]m.Hash{},
	}
	c.assignPresence()
	c.buildEntries(builder)
	if c.currentFolder().selectedId.Name == "" {
		c.selectFirst()
	}

	folder := c.currentFolder()
	screen := &w.Screen{
		CurrentPath:   c.currentPath,
		Progress:      c.progress(),
		SelectedId:    folder.selectedId,
		OffsetIdx:     folder.offsetIdx,
		SortColumn:    folder.sortColumn,
		SortAscending: folder.sortAscending,
	}

	screen.Entries = make([]w.File, len(c.entries))
	c.stats(screen)
	copy(screen.Entries, c.entries)
	return screen
}

func (c *controller) assignPresence() {
	c.update(func(file *w.File) {
		if presence, ok := c.presence[file.Hash]; ok {
			file.Presence = presence
		} else {
			file.Presence = w.Resolved
		}
	})
	c.update(func(file *w.File) {
		log.Printf("assignPresence: file %s", file)
	})
}

func (c *controller) buildEntries(builder *screenBuilder) {
	c.entries = c.entries[:0]

	c.handleOrigin(builder, c.archives[c.origin])

	for _, root := range c.roots[1:] {
		c.handleCopy(builder, c.archives[root])
	}

	c.sort()
}

func (c *controller) handleOrigin(builder *screenBuilder, archive *archive) {
	for _, file := range archive.files {
		if file.Path == c.currentPath {
			c.entries = append(c.entries, w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
				Hash:     file.Hash,
				Pending:  file.Pending,
				Presence: file.Presence,
			})
		} else if strings.HasPrefix(file.Path.String(), c.currentPath.String()) {
			relPath := file.Path
			if len(c.currentPath) > 0 {
				relPath = file.Path[len(c.currentPath)+1:]
			}
			name := m.Name(strings.SplitN(relPath.String(), "/", 2)[0])

			i, found := m.Find(c.entries, func(entry w.File) bool { return name == entry.Name })
			if found {
				c.entries[i].Size += file.Size
				if c.entries[i].ModTime.Before(file.ModTime) {
					c.entries[i].ModTime = file.ModTime
				}
				c.mergeStatus(&c.entries[i], file)
			} else {
				entry := w.File{
					FileMeta: m.FileMeta{
						FileId: m.FileId{
							Root: file.Root,
							Path: c.currentPath,
							Name: name,
						},
						Size:    file.Size,
						ModTime: file.ModTime,
					},
					FileKind: w.FileFolder,
					Pending:  file.Pending,
					Presence: file.Presence,
				}
				c.entries = append(c.entries, entry)
			}
		}
	}
	builder.originHashed = true
}

func (c *controller) mergeStatus(folder, file *w.File) {
	if folder.Presence < file.Presence {
		folder.Presence = file.Presence
	}
	if file.Pending {
		folder.Pending = true
	}
}

func (c *controller) handleCopy(builder *screenBuilder, archive *archive) {
	if !builder.originHashed {
		return
	}

	for _, file := range archive.files {
		if c.presence[file.Hash] != w.Absent {
			continue
		}
		if hash, ok := builder.copyNameHash[file.Name]; ok && file.Hash == hash {
			continue
		}
		if file.Path == c.currentPath {
			entry := w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
				Hash:     file.Hash,
				Pending:  file.Pending,
				Presence: file.Presence,
			}

			c.entries = append(c.entries, entry)
			builder.copyNameHash[entry.Name] = entry.Hash
		} else if strings.HasPrefix(file.Path.String(), c.currentPath.String()) {
			relPath := file.Path
			if len(c.currentPath) > 0 {
				relPath = file.Path[len(c.currentPath)+1:]
			}
			name := m.Name(strings.SplitN(relPath.String(), "/", 2)[0])

			_, found := m.Find(c.entries, func(entry w.File) bool { return name == entry.Name })
			if found {
				continue
			}
			entry := w.File{
				FileMeta: m.FileMeta{
					FileId: m.FileId{
						Root: file.Root,
						Path: c.currentPath,
						Name: name,
					},
				},
				FileKind: w.FileFolder,
				Pending:  file.Pending,
				Presence: file.Presence,
			}

			c.entries = append(c.entries, entry)

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

func (c *controller) stats(screen *w.Screen) {
	screen.PendingFiles, screen.DuplicateFiles, screen.AbsentFiles = 0, 0, 0

	c.update(func(file *w.File) {
		if file.Pending {
			screen.PendingFiles++
		}
	})

	for _, presence := range c.presence {
		switch presence {
		case w.Duplicate:
			screen.DuplicateFiles++
		case w.Absent:
			screen.AbsentFiles++
		}
	}
}
