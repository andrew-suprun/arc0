package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

type nameHashPair struct {
	m.Name
	m.Hash
}
type nameHashSet map[nameHashPair]struct{}

func (c *controller) buildView() *w.View {
	nameHashes := nameHashSet{}
	c.populateEntries(nameHashes)

	folder := c.currentFolder()

	nEntries := len(c.view.Entries)
	if nEntries > 0 {
		if folder.selectedId.Base == "" {
			folder.selectedId = c.view.Entries[0].Id
		} else {
			found := false
			for idx, entry := range c.view.Entries {
				if entry.Id == folder.selectedId {
					c.selectedIdx = idx
					found = true
					break
				}
			}
			if !found {
				if c.selectedIdx >= nEntries {
					c.selectedIdx = nEntries - 1
				}
				if c.selectedIdx < 0 {
					c.selectedIdx = 0
				}
				folder.selectedId = c.view.Entries[c.selectedIdx].Id
			}
		}
	}

	c.view.CurrentPath = c.currentPath
	c.view.Progress = c.progress()
	c.view.SelectedId = folder.selectedId
	c.view.OffsetIdx = folder.offsetIdx
	c.view.SortColumn = folder.sortColumn
	c.view.SortAscending = folder.sortAscending

	c.stats()
	return &c.view
}

func (c *controller) populateEntries(nameHashes nameHashSet) {
	c.view.Entries = c.view.Entries[:0]
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
		nameHash := nameHashPair{Name: file.Name, Hash: file.Hash}

		if file.Path == c.currentPath {
			if _, ok := nameHashes[nameHash]; !ok {
				c.view.Entries = append(c.view.Entries, &w.File{
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

			i, found := m.Find(c.view.Entries, func(entry *w.File) bool {
				return name == entry.Base && entry.Kind == w.FileFolder
			})
			if found {
				c.view.Entries[i].Size += file.Size
				if c.view.Entries[i].ModTime.Before(file.ModTime) {
					c.view.Entries[i].ModTime = file.ModTime
				}
				if c.view.Entries[i].State < state {
					c.view.Entries[i].State = state
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
				c.view.Entries = append(c.view.Entries, entry)
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
			Root:          c.origin,
			Tab:           " Copying",
			Value:         float64(c.totalCopiedSize+uint64(c.fileCopiedSize)) / float64(c.copySize),
			Speed:         c.copySpeed,
			TimeRemaining: c.timeRemaining,
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
		infos = append(infos, w.ProgressInfo{
			Root:          root,
			Tab:           tab,
			Value:         value,
			Speed:         archive.speed,
			TimeRemaining: archive.timeRemaining,
		})
	}
	return infos
}

func (c *controller) stats() {
	c.view.PendingFiles, c.view.DuplicateFiles, c.view.AbsentFiles = 0, 0, 0

	for _, presence := range c.state {
		switch presence {
		case w.Pending:
			c.view.PendingFiles++
		case w.Duplicate:
			c.view.DuplicateFiles++
		case w.Absent:
			c.view.AbsentFiles++
		}
	}
	if c.archives[c.origin].progressState == m.Initial {
		c.view.AbsentFiles = 0
	}
}
