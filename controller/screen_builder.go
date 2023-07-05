package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

func (c *controller) buildScreen() *w.Screen {
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
	for i := range c.entries {
		screen.Entries[i] = *c.entries[i]
	}

	for root, archive := range c.archives {
		if root == c.origin {
			c.handleOrigin(archive, screen)
		} else {
			c.handleCopy(archive, screen)
		}
	}
	return screen
}

func (c *controller) handleOrigin(archive *archive, screen *w.Screen) {
	for _, file := range archive.infoByName {
		if file.Path == c.currentPath {
			c.entries = append(c.entries, &w.File{
				FileMeta: file.FileMeta,
				FileKind: w.FileRegular,
			})
		} else if strings.HasPrefix(file.Path.String(), c.currentPath.String()) {
			relPath := c.currentPath[len(file.Path)+1:]
			name := m.Name(strings.SplitN(relPath.String(), "/", 2)[0])
			hasName := false
			for i := range c.entries {
				if name == c.entries[i].Name {
					hasName = true
					c.entries[i].Size += file.Size
					if c.entries[i].ModTime.Before(file.ModTime) {
						c.entries[i].ModTime = file.ModTime
					}
					break
				}
			}
			if !hasName {
				c.entries = append(c.entries, &w.File{
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
				})
			}
		}
	}
}

func (c *controller) handleCopy(archive *archive, screen *w.Screen) {
}

func (c *controller) progress() []w.ProgressInfo {
	return nil
}
