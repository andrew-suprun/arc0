package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	archive := c.archives[tree.Root]
	for _, meta := range tree.FileMetas {
		c.files[meta.Id] = &w.File{FileMeta: meta}
		archive.totalSize += meta.Size
	}

	c.archives[tree.Root].progressState = m.Scanned
	for _, archive := range c.archives {
		if archive.progressState != m.Scanned {
			return
		}
	}
	for _, archive := range c.archives {
		archive.scanner.Send(m.HashArchive{})
	}
}

func (c *controller) archiveHashed(tree m.ArchiveHashed) {
	archive := c.archives[tree.Root]
	archive.progressState = m.Hashed
	archive.totalSize, archive.totalSize = 0, 0
	for _, archive := range c.archives {
		if archive.progressState != m.Hashed {
			return
		}
	}

	files := map[m.Hash][]*w.File{}
	for _, file := range c.files {
		if file.Root == c.origin {
			files[file.Hash] = append(files[file.Hash], file)
		}
	}
	for _, entries := range files {
		if len(entries) == 1 {
			c.keepFile(entries[0])
		}
	}
}

func (c *controller) fileHashed(hashed m.FileHashed) {
	archive := c.archives[hashed.Root]
	file := c.files[hashed.Id]
	file.Hash = hashed.Hash
	archive.totalHashed += file.Size
	archive.hashingProgress.Hashed = 0

	hashes := map[m.Hash]struct{}{}
	bySize := []*w.File{}

	for _, entry := range c.files {
		if entry.Size == file.Size {
			if entry.Hash == "" {
				return
			}
			bySize = append(bySize, entry)
			hashes[entry.Hash] = struct{}{}
		}
	}

	for hash := range hashes {
		byHash := []*w.File{}
		for _, entry := range bySize {
			if entry.Hash == hash {
				byHash = append(byHash, entry)
			}
		}

		entriesByArchive := map[m.Root][]*w.File{}
		for _, entry := range byHash {
			entriesByArchive[entry.Root] = append(entriesByArchive[entry.Root], entry)
		}
		originEntries := entriesByArchive[c.origin]

		if len(originEntries) == 0 {
			c.state[hash] = w.Absent
		} else if len(originEntries) > 1 {
			c.state[hash] = w.Duplicate
		}
	}
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].hashingProgress = event
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.copyingProgress = event
	log.Printf("handleCopyingProgress: copied: %d", event.Copied)
}

func (c *controller) filesHandled(event m.FilesHandled) {
	log.Printf("filesHandled: %v", event)
	delete(c.state, event.Hash)

	if event.Copy != nil {
		copy := event.Copy
		if entry, ok := c.files[copy.From]; ok {
			c.totalCopied += entry.Size
			c.copyingProgress.Copied = 0
			if c.copySize == c.totalCopied {
				c.copySize, c.totalCopied = 0, 0
			}
			log.Printf("filesHandled: c.copySize: %d, c.totalCopied: %d", c.copySize, c.totalCopied)
		} else {
			log.Printf("### filesHandled: not found copied: %#v", event.Copy)
		}
	}
}

func (c *controller) makeSelectedVisible() {
	selectedIdx := c.selectedIdx()
	offsetIdx := c.currentFolder().offsetIdx

	if offsetIdx > selectedIdx {
		offsetIdx = selectedIdx
	}
	if offsetIdx < selectedIdx+1-c.feedback.FileTreeLines {
		offsetIdx = selectedIdx + 1 - c.feedback.FileTreeLines
	}

	c.currentFolder().offsetIdx = offsetIdx
}

func (c *controller) selectedIdx() int {
	selectedId := c.currentFolder().selectedId
	if idx, found := m.Find(c.entries, func(entry w.File) bool { return entry.Id == selectedId }); found {
		return idx
	}

	log.Panicf("selectedIdx filed")
	return 0
}
