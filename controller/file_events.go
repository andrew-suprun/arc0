package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) archiveFiles(event m.ArchiveFiles) {
	archive := c.archives[event.Root]
	for _, file := range event.Files {
		c.byId[file.Id] = file
		c.bySize[file.Size] = append(c.bySize[file.Size], file)
		archive.totalSize += file.Size
	}
}

func (c *controller) fileScanned(event m.FileScanned) {
	c.byHash[event.Hash] = append(c.byHash[event.Hash], event.File)
	archive := c.archives[event.Root]
	archive.totalHashed += event.File.Size
	archive.fileHashed = 0
}

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	archive := c.archives[tree.Root]
	archive.progressState = m.Scanned
	for _, archive := range c.archives {
		if archive.progressState != m.Scanned {
			return
		}
	}
	c.archivesScanned = true
	c.autoresolve()
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].fileHashed = event.Hashed
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.fileCopiedSize = uint64(event)
}

func (c *controller) fileDeleted(event m.FileDeleted) {
	log.Printf("### %s", event)
	c.state[event.Hash] = w.Resolved
}

func (c *controller) fileRenamed(event m.FileRenamed) {
	log.Printf("### %s", event)
	c.state[event.Hash] = w.Resolved
}

func (c *controller) fileCopied(event m.FileCopied) {
	log.Printf("### %s", event)
	c.state[event.Hash] = w.Resolved
	c.fileCopiedSize = 0
	file := c.byHash[event.Hash][0]
	c.totalCopiedSize += file.Size
	if c.totalCopiedSize == c.copySize {
		c.totalCopiedSize, c.copySize = 0, 0
	}
}
