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
			for root, entries := range entriesByArchive {
				if root != c.origin {
					for _, entry := range entries {
						entry.Status = w.Absent
					}
				}
			}
		} else if len(originEntries) == 1 {
			c.keepFile(originEntries[0])
		} else {
			for _, entry := range originEntries {
				entry.Status = w.Duplicate
				log.Printf("^^^ dup: %s", entry)
			}
		}
	}
}

func (c *controller) handleProgress(event m.Progress) {
	archive := c.archives[event.Root]
	archive.progress = event

	if event.ProgressState == m.FileTreeHashed {
		archive.progressState = m.FileTreeHashed
	}
}

func (c *controller) fileRenamed(renamed m.FileRenamed) {
	log.Printf("Event %v", renamed)
	// TODO

	// c.removeFolderFile(renamed.FileId)
}

func (c *controller) fileDeleted(deleted m.FileDeleted) {
	log.Printf("Event %v", deleted)
	c.removeFolderFile(m.FileId(deleted))
}

func (c *controller) fileCopied(copied m.FileCopied) {
	log.Printf("Event %v", copied)
	fromArchive := c.archives[copied.From.Root]
	file := fromArchive.files[copied.From.FullName()]
	file.Status = w.Resolved

	toArchive := c.archives[copied.To]
	toArchive.totalCopied += file.Size
	toArchive.progress.HandledSize = 0
	if toArchive.copySize == toArchive.totalCopied {
		toArchive.copySize, toArchive.totalCopied = 0, 0
	}
	log.Printf("### fileCopied: root: %q, copy size: %d, totalCopied: %d", copied.To, toArchive.copySize, toArchive.totalCopied)
}

func (c *controller) makeSelectedVisible() {
	selectedIdx := c.selectedIdx()
	offsetIdx := c.folders[c.currentPath].offsetIdx

	if offsetIdx > selectedIdx {
		offsetIdx = selectedIdx
	}
	if offsetIdx < selectedIdx+1-c.feedback.FileTreeLines {
		offsetIdx = selectedIdx + 1 - c.feedback.FileTreeLines
	}

	c.folders[c.currentPath].offsetIdx = offsetIdx
}

func (c *controller) removeFolderFile(id m.FileId) {
	// archive := c.archives[id.Root]
	// delete(archive.files, id.FullName())
}

func (c *controller) selectedIdx() int {
	selectedId := c.folders[c.currentPath].selectedId
	if idx, found := m.Find(c.entries, func(entry w.File) bool { return entry.FileId == selectedId }); found {
		return idx
	}

	log.Panicf("selectedIdx filed")
	return 0
}
