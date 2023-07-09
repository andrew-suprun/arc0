package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	archive := c.archives[tree.Root]
	for _, meta := range tree.FileMetas {
		archive.files[meta.FullName()] = &w.File{FileMeta: meta}
		archive.totalSize += meta.Size
	}

	c.archives[tree.Root].progressState = m.FileTreeScanned
	for _, archive := range c.archives {
		if archive.progressState != m.FileTreeScanned {
			return
		}
	}
	for _, archive := range c.archives {
		archive.scanner.Send(m.HashArchive{})
	}
}

func (c *controller) fileHashed(hashed m.FileHashed) {
	log.Printf("Event %v", hashed)
	archive := c.archives[hashed.Root]
	file := archive.fileByFullName(hashed.FullName())
	file.Hash = hashed.Hash
	archive.totalHashed += file.Size
	archive.progress.HandledSize = 0

	hashes := map[m.Hash]struct{}{}
	bySize := []*w.File{}
	for _, archive := range c.archives {
		for _, entry := range archive.files {
			if entry.Size == file.Size {
				if entry.Hash == "" {
					return
				}
				bySize = append(bySize, entry)
				hashes[entry.Hash] = struct{}{}
			}
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
			c.hashStatuses[hash] = w.Absent
		} else if len(originEntries) == 1 {
			c.keepFile(originEntries[0])
		} else {
			c.hashStatuses[hash] = w.Duplicate
		}
	}
}

func (c *controller) handleProgress(event m.Progress) {
	root := c.origin
	if event.ProgressState != m.CopyingFile {
		root = event.Root
	}
	archive := c.archives[root]
	archive.progress = event

	if event.ProgressState == m.FileTreeHashed {
		archive.progressState = m.FileTreeHashed
	}
}

func (c *controller) filesHandled(event m.FilesHandled) {
	log.Printf("Event %v", event)
	c.hashStatuses[event.Hash] = w.Resolved

	for _, deleted := range event.Delete {
		archive := c.archives[deleted.Root]
		if pending, ok := archive.pending[deleted.FullName()]; ok {
			delete(archive.files, pending.FullName())
			delete(archive.pending, pending.PendingName)
		} else {
			log.Printf("### filesHandled: not found deleted: %s", deleted)
		}
	}

	for _, renamed := range event.Rename {
		archive := c.archives[renamed.Root]
		if pending, ok := archive.pending[renamed.FullName()]; ok {
			delete(archive.files, pending.FullName())
			delete(archive.pending, pending.PendingName)

			pending.Path = renamed.NewFullName.Path
			pending.Name = renamed.NewFullName.Name
			pending.PendingName = m.FullName{}
			archive.files[pending.FullName()] = pending
		} else {
			log.Printf("### filesHandled: not found renamed: %s", renamed)
		}
	}

	if event.Copy != nil {
		if pending, ok := c.archives[event.Copy.From.Root].pending[event.Copy.From.FullName()]; ok {
			origin := c.archives[c.origin]
			origin.totalCopied += pending.Size
			origin.progress.HandledSize = 0
			if origin.copySize == origin.totalCopied {
				origin.copySize, origin.totalCopied = 0, 0
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
	if idx, found := m.Find(c.entries, func(entry w.File) bool { return entry.FileId == selectedId }); found {
		return idx
	}

	log.Panicf("selectedIdx filed")
	return 0
}
