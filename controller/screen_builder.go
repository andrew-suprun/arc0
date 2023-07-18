package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

type nameHashPair struct {
	m.Base
	m.Hash
}
type nameHashSet map[nameHashPair]struct{}

func (c *controller) buildScreen() {
	nameHashes := nameHashSet{}
	c.populateEntries(nameHashes)

	folder := c.currentFolder()
	c.screen.CurrentPath = c.currentPath
	c.screen.Progress = c.progress()
	c.screen.SelectedId = c.selectedId()
	c.screen.OffsetIdx = folder.offsetIdx
	c.screen.SortColumn = folder.sortColumn
	c.screen.SortAscending = folder.sortAscending

	if c.screen.SelectedId.Base == "" && len(c.screen.Entries) > 0 {
		c.screen.SelectedId = c.screen.Entries[0].Id
	}

	c.stats()
}

func (c *controller) populateEntries(nameHashes nameHashSet) {
	c.screen.Entries = c.screen.Entries[:0]
	for hash, files := range c.files {
		state := c.calcState(hash, files)
		c.state[hash] = state
		c.addEntries(state, files, nameHashes)
	}
	c.sort()
}

func (c *controller) calcState(hash m.Hash, files []*m.File) w.State {
	state := c.state[hash]
	if state == w.Pending {
		return w.Pending
	}
	originFiles := 0
	for _, file := range files {
		if file.Root == c.origin {
			originFiles++
		}
	}
	if originFiles == 0 {
		return w.Absent
	} else if originFiles > 1 {
		return w.Duplicate
	}
	return w.Resolved
}

func (c *controller) addEntries(state w.State, files []*m.File, nameHashes nameHashSet) {
	originProgressState := c.archives[c.origin].progressState
	for _, file := range files {
		if file.Root != c.origin && (originProgressState == m.Initial || state != w.Absent) {
			continue
		}
		nameHash := nameHashPair{Base: file.Base, Hash: file.Hash}

		if file.Path == c.currentPath {
			if _, ok := nameHashes[nameHash]; !ok {
				c.screen.Entries = append(c.screen.Entries, &w.File{
					File:  *file,
					Kind:  w.FileRegular,
					State: state,
				})
			}
		} else if strings.HasPrefix(file.Path.String(), c.currentPath.String()) {
			relPath := file.Path
			if len(c.currentPath) > 0 {
				relPath = file.Path[len(c.currentPath)+1:]
			}
			name := m.Base(strings.SplitN(relPath.String(), "/", 2)[0])

			i, found := m.Find(c.screen.Entries, func(entry *w.File) bool {
				return name == entry.Base && entry.Kind == w.FileFolder
			})
			if found {
				c.screen.Entries[i].Size += file.Size
				if c.screen.Entries[i].ModTime.Before(file.ModTime) {
					c.screen.Entries[i].ModTime = file.ModTime
				}
				if c.screen.Entries[i].State < state {
					c.screen.Entries[i].State = state
				}
			} else {
				entry := &w.File{
					File: m.File{
						Id: m.Id{
							Name: m.Name{
								Path: c.currentPath,
								Base: name,
							},
						},
						Size:    file.Size,
						ModTime: file.ModTime,
					},
					Kind:  w.FileFolder,
					State: state,
				}
				c.screen.Entries = append(c.screen.Entries, entry)
			}
		}
		nameHashes[nameHash] = struct{}{}
	}
}

func (c *controller) progress() []w.ProgressInfo {
	infos := []w.ProgressInfo{}
	archive := c.archives[c.origin]
	if archive.progressState == m.Scanned && c.copySize > 0 {
		infos = append(infos, w.ProgressInfo{
			Root:  c.origin,
			Tab:   " Copying",
			Value: float64(c.totalCopied+uint64(c.fileCopied)) / float64(c.copySize),
		})
	}
	var tab string
	var value float64
	for _, root := range c.roots {
		archive := c.archives[root]
		if archive.progressState == m.Scanned {
			continue
		}
		tab = " Hashing"
		value = float64(archive.totalHashed+archive.fileHashed) / float64(archive.totalSize)
		infos = append(infos, w.ProgressInfo{Root: root, Tab: tab, Value: value})
	}
	return infos
}

func (c *controller) stats() {
	c.screen.PendingFiles, c.screen.DuplicateFiles, c.screen.AbsentFiles = 0, 0, 0

	for _, presence := range c.state {
		switch presence {
		case w.Pending:
			c.screen.PendingFiles++
		case w.Duplicate:
			c.screen.DuplicateFiles++
		case w.Absent:
			c.screen.AbsentFiles++
		}
	}
	if c.archives[c.origin].progressState == m.Initial {
		c.screen.AbsentFiles = 0
	}
}
