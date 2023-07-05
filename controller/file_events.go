package controller

import (
	m "arch/model"
	w "arch/widgets"
	"path/filepath"
)

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	archive := c.archives[tree.Root]
	for _, meta := range tree.FileMetas {
		file := &w.File{FileMeta: meta}
		archive.infoByName[meta.FullName()] = file
		archive.infosBySize[file.Size][file] = struct{}{}
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
	file := c.archives[fileHash.Root].infoByName[fileHash.FullName()]
	file.Hash = fileHash.Hash
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
}

func (c *controller) removeFolderFile(id m.FileId) {
	archive := c.archives[id.Root]
	file := archive.infoByName[id.FullName()]
	delete(archive.infoByName, id.FullName())
	delete(archive.infosBySize[file.Size], file)
}

func (c *controller) scanProgress(event m.ScanProgress) {
	c.archives[event.Root].progress = event

	if event.ProgressState == m.FileTreeHashed {
		for _, archive := range c.archives {
			if archive.progress.ProgressState != m.FileTreeHashed {
				return
			}
		}
		c.autoResolve()
	}
}

func (c *controller) autoResolve() {
	archive := c.archives[c.origin]
	for _, file := range archive.infoByName {
		if file.Status == w.Pending {
			c.keepFile(file)
		}
	}
}

func (c *controller) fileCopyProgress(event m.FileCopyProgress) {
	archive := c.archives[c.origin]
	archive.fileCopied = uint64(event)
}

func dir(path m.Path) m.Path {
	path = m.Path(filepath.Dir(path.String()))
	if path == "." {
		return ""
	}
	return path
}

func name(path m.Path) m.Name {
	return m.Name(filepath.Base(path.String()))
}

func (c *controller) selectedIdx() int {
	selectedId := c.folders[c.currentPath].selectedId
	for idx, entry := range c.entries {
		if entry.FileId == selectedId {
			return idx
		}
	}
	return 0
}
