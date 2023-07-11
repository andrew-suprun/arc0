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
}

func (c *controller) fileHashed(hashed m.FileHashed) {
	log.Printf("Event %v", hashed)
	archive := c.archives[hashed.Root]
	file := c.fileById(hashed.Id)
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
			c.presence[hash] = w.Absent
		} else if len(originEntries) == 1 {
			c.keepFile(originEntries[0])
		} else {
			c.presence[hash] = w.Duplicate
		}
	}
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].hashingProgress = event
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.archives[c.origin].copyingProgress = event
}

func (c *controller) filesHandled(event m.FilesHandled) {
	log.Printf("Event %v", event)
	c.presence[event.Hash] = w.Resolved

	for _, deleted := range event.Delete {
		if pending, ok := c.pending[deleted]; ok {
			delete(c.files, pending.Id)
			delete(c.pending, pending.PendingId)
		} else {
			log.Printf("### filesHandled: not found deleted: %s", deleted)
		}
	}

	for _, renamed := range event.Rename {
		if pending, ok := c.pending[renamed.Id]; ok {
			delete(c.files, pending.Id)
			delete(c.pending, pending.NewId())

			pending.Id = m.Id{Root: renamed.Root, Name: renamed.NewName}
			pending.PendingId = m.Id{}
			pending.Pending = false
			c.files[pending.Id] = pending
		} else {
			log.Printf("### filesHandled: not found renamed: %s", renamed)
		}
	}

	if event.Copy != nil {
		copy := event.Copy
		if pending, ok := c.pending[copy.From]; ok {
			origin := c.archives[c.origin]
			origin.totalCopied += pending.Size
			origin.copyingProgress.Copied = 0
			if origin.copySize == origin.totalCopied {
				origin.copySize, origin.totalCopied = 0, 0
			}
			pending.Pending = false
			pending.PendingId = m.Id{}
			for _, root := range copy.To {
				newFile := &w.File{
					FileMeta: pending.FileMeta,
					FileKind: pending.FileKind,
					Hash:     pending.Hash,
				}
				to := m.Id{Root: root, Name: newFile.Name}
				c.files[to] = newFile
			}
		} else {
			log.Printf("### filesHandled: not found copied: %s", event.Copy)
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
