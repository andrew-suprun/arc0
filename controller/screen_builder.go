package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
	"strings"
)

func (c *controller) buildScreen() *w.Screen {
	log.Printf("### build screen ----")
	c.buildEntries()

	folder := c.folders[c.currentPath]
	screen := &w.Screen{
		CurrentPath:    c.currentPath,
		Progress:       c.progress(),
		SelectedId:     folder.selectedId,
		OffsetIdx:      folder.offsetIdx,
		SortColumn:     folder.sortColumn,
		SortAscending:  folder.sortAscending,
		PendingFiles:   c.pendingFiles,
		DuplicateFiles: c.duplicateFiles,
		AbsentFiles:    c.absentFiles,
	}

	screen.Entries = make([]w.File, len(c.entries))
	copy(screen.Entries, c.entries)
	return screen
}

func (c *controller) buildEntries() {
	c.pendingFiles, c.duplicateFiles, c.absentFiles = 0, 0, 0
	c.entries = c.entries[:0]
	for root, archive := range c.archives {
		if root == c.origin {
			c.handleOrigin(archive)
		} else {
			c.handleCopy(archive)
		}
	}
	c.sort()
}

func (c *controller) handleOrigin(archive *archive) {
	hashCounts := map[m.Hash]int{}
	for _, file := range archive.infoByName {
		if file.Hash != "" && file.Root == c.origin {
			hashCounts[file.Hash]++
		}
	}
	for _, count := range hashCounts {
		if count > 1 {
			c.duplicateFiles++
		}
	}
	for _, file := range archive.infoByName {
		if file.Path == c.currentPath {
			entry := w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
			}
			if hashCounts[file.Hash] > 1 {
				entry.Status = w.Duplicate
			}
			c.entries = append(c.entries, entry)
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
				if hashCounts[file.Hash] > 1 {
					c.entries[i].Status = w.Duplicate
				}
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
				}
				if hashCounts[file.Hash] > 1 {
					entry.Status = w.Duplicate
				}
				c.entries = append(c.entries, entry)
			}
		}
	}
}

func (c *controller) handleCopy(archive *archive) {
	// TODO
}

func (c *controller) progress() []w.ProgressInfo {
	// TODO
	return nil
}
