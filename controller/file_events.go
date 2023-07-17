package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) totalSize(event m.TotalSize) {
	c.archives[event.Root].totalSize += event.Size
}

func (c *controller) fileScanned(event m.FileScanned) {
	c.files[event.Hash] = append(c.files[event.Hash], event.File)
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
	c.autoResolve()
}

func (c *controller) autoResolve() {
	c.archivesScanned = true
	for _, files := range c.files {
		originFiles := []*m.File{}
		names := map[m.Name]struct{}{}
		for _, file := range files {
			if file.Root == c.origin {
				originFiles = append(originFiles, file)
			}
			names[file.Name] = struct{}{}
		}
		if len(originFiles) == 1 && (len(files) != len(c.roots) || len(names) != 1) {
			c.keepFile(originFiles[0])
		}
	}
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].fileHashed = event.Hashed
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.fileCopied = uint64(event)
}

func (c *controller) filesHandled(event m.FilesHandled) {
	log.Printf("filesHandled: %s", event)
	c.state[event.Hash] = w.Resolved
	if event.Copy != nil {
		c.fileCopied = 0
		file := c.find(func(entry *m.File) bool {
			return entry.Id == m.Id(event.Copy.From)
		})
		c.totalCopied += file.Size
	}
	if c.totalCopied == c.copySize {
		c.totalCopied, c.copySize = 0, 0
	}
}
