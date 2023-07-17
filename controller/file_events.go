package controller

import (
	m "arch/model"
	"log"
)

func (c *controller) fileScanned(event m.FileScanned) {
	c.files[event.Hash] = append(c.files[event.Hash], event.File)
	log.Printf("fileScanned: file: %s", event.File)
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
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].hashingProgress = event
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.copyingProgress = event
}

func (c *controller) filesHandled(event m.FilesHandled) {
	c.filesHandledEvents = append(c.filesHandledEvents, event)
}
