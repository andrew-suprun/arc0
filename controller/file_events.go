package controller

import (
	m "arch/model"
	"log"
)

func (c *controller) archiveScanned(event m.ArchiveScanned) {
	if event.Root == c.origin {
		c.addEntries(event)
	}

	archive := c.archives[event.Root]
	for _, file := range event.Files {
		archive.totalSize += file.Size
	}

	allScanned := true
	for _, archive := range c.archives {
		if archive.progressState != m.ProgressScanned {
			allScanned = false
			break
		}
	}

	if allScanned {
		for _, archive := range c.archives {
			archive.scanner.Send(m.HashArchive{})
		}
	}
}

func (c *controller) addEntries(event m.ArchiveScanned) {
	for _, file := range event.Files {
		entry := &m.File{
			Meta:  file,
			Kind:  m.FileRegular,
			State: m.Initial,
		}
		c.byId[file.Id] = entry
		entries := c.bySize[file.Size]
		entries = append(entries, entry)
		c.bySize[file.Size] = entries
		c.addEntry(entry)
	}
	c.setInitialSelection("")
	c.currentPath = ""
}

func (c *controller) addEntry(entry *m.File) {
	log.Printf("addEntry: entry: %q", entry.Id)
	folder := c.getFolder(entry.Path)
	folder.entries[entry.Base] = entry
	name := entry.ParentName()
	for name.Base != "." {
		log.Printf("addEntry: parent path: %q, name: %q", name.Path, name.Base)
		parentFolder := c.getFolder(name.Path)
		item := parentFolder.entries[name.Base]
		if item != nil {
			if item.Kind != m.FileFolder {
				log.Panicf("ERROR: Name collision in controller.addEntry()")
			}
			item.Size += entry.Size
			if item.ModTime.Before(entry.ModTime) {
				item.ModTime = entry.ModTime
			}
		} else {
			folderEntry := &m.File{
				Meta: m.Meta{
					Id: m.Id{
						Root: entry.Root,
						Name: name,
					},
					Size:    entry.Size,
					ModTime: entry.ModTime,
				},
				Kind:  m.FileFolder,
				State: m.Initial,
			}
			c.addEntry(folderEntry)
		}

		name = name.Path.ParentName()
	}
}

func (c *controller) setInitialSelection(path m.Path) {
	c.currentPath = path
	c.selectFirst()
	for _, entry := range c.currentFolder().entries {
		if entry.Kind == m.FileFolder {
			c.setInitialSelection(m.Path(entry.Name.String()))
		}
	}
}

func (c *controller) removeEntry(id m.Id) {
	log.Panic("Implement controller.remove()")
	c.updateFolderStates("")
}

func (c *controller) fileHashed(event m.FileHashed) {
	file := c.byId[event.Id]
	file.Hash = event.Hash

	archive := c.archives[event.Root]
	archive.totalHashed += file.Size
	archive.fileHashed = 0

	hashes := map[m.Hash]struct{}{}
	files := c.bySize[file.Size]
	for _, entry := range files {
		hashes[entry.Hash] = struct{}{}
	}
	for hash := range hashes {
		var entries []*m.File
		var origins []*m.File
		names := map[m.Name]struct{}{}
		for _, entry := range files {
			if entry.Hash == hash {
				entries = append(entries, entry)
				names[entry.Name] = struct{}{}
				if entry.Root == c.origin {
					origins = append(origins, entry)
				}
			}
		}
		switch len(origins) {
		case 0:
			for _, entry := range entries {
				entry.State = m.Absent
			}

		case 1:
			c.keepFile(origins[0])

		default:
			for _, entry := range entries {
				entry.State = m.Duplicate
			}
		}
	}
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].fileHashed = event.Hashed
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.fileCopiedSize = uint64(event)
}

func (c *controller) fileDeleted(event m.FileDeleted) {
	log.Printf("### %s", event)
	c.pendingFiles--
}

func (c *controller) fileRenamed(event m.FileRenamed) {
	log.Printf("### %s", event)
	c.setState(event.Hash, m.Resolved)
	c.pendingFiles--
}

func (c *controller) fileCopied(event m.FileCopied) {
	log.Printf("### %s", event)
	c.setState(event.Hash, m.Resolved)
	c.pendingFiles--
	c.fileCopiedSize = 0
	file := c.byId[event.From]
	c.totalCopiedSize += file.Size
	if c.totalCopiedSize == c.copySize {
		c.totalCopiedSize, c.copySize = 0, 0
	}
}

func (c *controller) setState(hash m.Hash, state m.State) {
	for _, entry := range c.byHash[hash] {
		entry.State = state
	}
	c.updateFolderStates("")
}

func (c *controller) updateFolderStates(path m.Path) m.State {
	state := m.Initial
	folder := c.getFolder(path)
	for _, entry := range folder.entries {
		if entry.Kind == m.FileFolder {
			state = mergeState(state, c.updateFolderStates(entry.Path))
		} else {
			state = mergeState(state, entry.State)
		}
	}
	return state
}

func mergeState(state1, state2 m.State) m.State {
	if state1 > state2 {
		return state1
	}
	return state2
}
