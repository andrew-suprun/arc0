package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

type screenBuilder struct {
	copyNameHash map[nameAndHash]struct{}
	originHashed bool
}

type nameAndHash struct {
	m.Base
	m.Hash
}

func (c *controller) buildScreen() *w.Screen {
	builder := &screenBuilder{
		copyNameHash: map[nameAndHash]struct{}{},
	}
	c.assignPresence()
	c.buildEntries(builder)
	if c.currentFolder().selectedId.Base == "" {
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
	for _, file := range c.files {
		if presence, ok := c.presence[file.Hash]; ok {
			file.Presence = presence
		} else {
			file.Presence = w.Resolved
		}
	}
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
	for _, file := range c.files {
		if file.Root != c.origin {
			continue
		}
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
			name := m.Base(strings.SplitN(relPath.String(), "/", 2)[0])

			i, found := m.Find(c.entries, func(entry w.File) bool { return name == entry.Base })
			if found {
				c.entries[i].Size += file.Size
				if c.entries[i].ModTime.Before(file.ModTime) {
					c.entries[i].ModTime = file.ModTime
				}
				c.mergeStatus(&c.entries[i], file)
			} else {
				entry := w.File{
					FileMeta: m.FileMeta{
						Id: m.Id{
							Root: file.Root,
							Name: m.Name{
								Path: c.currentPath,
								Base: name,
							},
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

	for _, file := range c.files {
		if file.Root == c.origin {
			continue
		}
		if c.presence[file.Hash] != w.Absent {
			continue
		}
		nameHash := nameAndHash{Base: file.Base, Hash: file.Hash}
		if _, ok := builder.copyNameHash[nameHash]; ok {
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
			builder.copyNameHash[nameHash] = struct{}{}
		} else if strings.HasPrefix(file.Path.String(), c.currentPath.String()) {
			relPath := file.Path
			if len(c.currentPath) > 0 {
				relPath = file.Path[len(c.currentPath)+1:]
			}
			name := m.Base(strings.SplitN(relPath.String(), "/", 2)[0])

			_, found := m.Find(c.entries, func(entry w.File) bool { return name == entry.Base })
			if found {
				continue
			}
			entry := w.File{
				FileMeta: m.FileMeta{
					Id: m.Id{
						Root: file.Root,
						Name: m.Name{
							Path: c.currentPath,
							Base: name,
						},
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
	archive := c.archives[c.origin]
	if archive.progressState == m.Hashed {
		infos = append(infos, w.ProgressInfo{
			Root:  c.origin,
			Tab:   " Copying",
			Value: float64(archive.totalCopied+archive.copyingProgress.Copied) / float64(archive.copySize),
		})
	}
	var tab string
	var value float64
	for _, root := range c.roots {
		archive := c.archives[root]
		if archive.progressState != m.Scanned {
			continue
		}
		tab = " Hashing"
		value = float64(archive.totalHashed+archive.hashingProgress.Hashed) / float64(archive.totalSize)
		infos = append(infos, w.ProgressInfo{Root: root, Tab: tab, Value: value})
	}
	return infos
}

func (c *controller) stats(screen *w.Screen) {
	screen.PendingFiles, screen.DuplicateFiles, screen.AbsentFiles = 0, 0, 0

	// TODO Change to c.pending
	for _, file := range c.files {
		if file.Pending {
			screen.PendingFiles++
		}
	}

	for _, presence := range c.presence {
		switch presence {
		case w.Duplicate:
			screen.DuplicateFiles++
		case w.Absent:
			screen.AbsentFiles++
		}
	}
}
