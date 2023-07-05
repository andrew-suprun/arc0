package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	archive := c.archives[tree.Root]
	for _, meta := range tree.FileMetas {
		file := &w.File{FileMeta: meta}
		archive.infoByName[meta.FullName()] = file

		bySize := archive.infosBySize[file.Size]
		if bySize == nil {
			bySize = map[*w.File]struct{}{file: {}}
			archive.infosBySize[file.Size] = bySize
		} else {
			bySize[file] = struct{}{}
		}
		archive.totalSize += meta.Size
	}

	c.archives[tree.Root].progress.ProgressState = m.FileTreeScanned
	for _, archive := range c.archives {
		if archive.progress.ProgressState != m.FileTreeScanned {
			return
		}
	}
	for _, archive := range c.archives {
		archive.scanner.Send(m.HashArchive{})
	}
}

func (c *controller) fileHashed(fileHash m.FileHashed) {
	archive := c.archives[fileHash.Root]
	file := archive.infoByName[fileHash.FullName()]
	file.Hash = fileHash.Hash
	archive.totalHandled += file.Size
	archive.progress.HandledSize = 0
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

func (c *controller) fileRenamed(renamed m.FileRenamed) {
	c.removeFolderFile(renamed.FileId)

	archive := c.archives[renamed.Root]
	file := archive.infoByName[renamed.FullName()]
	archive.infoByName[renamed.NewFullName] = file
}

func (c *controller) fileDeleted(deleted m.FileDeleted) {
	c.removeFolderFile(m.FileId(deleted))
}

func (c *controller) fileCopied(copied m.FileCopied) {
	fromArchive := c.archives[copied.From.Root]
	file := fromArchive.infoByName[copied.From.FullName()]

	toArchive := c.archives[copied.To]
	toArchive.infoByName[copied.From.FullName()] = file
	toArchive.totalHandled += file.Size
	toArchive.progress.HandledSize = 0
	if toArchive.totalSize == fromArchive.totalHandled {
		toArchive.totalSize, fromArchive.totalHandled = 0, 0
	}
}

func (c *controller) removeFolderFile(id m.FileId) {
	archive := c.archives[id.Root]
	file := archive.infoByName[id.FullName()]
	delete(archive.infoByName, id.FullName())
	delete(archive.infosBySize[file.Size], file)
}

func (c *controller) handleProgress(event m.Progress) {
	archive := c.archives[event.Root]
	archive.progress = event

	if event.ProgressState == m.FileTreeHashed {
		archive.totalSize = 0
		for _, archive := range c.archives {
			if archive.progress.ProgressState != m.FileTreeHashed {
				return
			}
		}
	}
	c.autoResolve()
}

func (c *controller) autoResolve() {
	archive := c.archives[c.origin]
	for _, file := range archive.infoByName {
		if file.Status == w.Pending {
			c.keepFile(file)
		}
	}
}

func (c *controller) selectedIdx() int {
	selectedId := c.folders[c.currentPath].selectedId
	if idx, found := m.Find(c.entries, func(entry w.File) bool { return entry.FileId == selectedId }); found {
		return idx
	}

	log.Panicf("selectedIdx filed")
	return 0
}
