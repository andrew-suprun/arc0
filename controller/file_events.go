package controller

import (
	m "arch/model"
	"log"
	"path/filepath"
	"time"
)

func (c *controller) archiveScanned(tree m.ArchiveScanned) {
	for _, meta := range tree.Metas {
		c.fileScanned(meta)
	}
	c.archives[tree.Root].progress.ProgressState = m.FileTreeScanned
	for _, archive := range c.archives {
		if archive.progress.ProgressState != m.FileTreeScanned {
			return
		}
	}
	for _, archive := range c.archives {
		archive.scanner.HashArchive()
	}
}

func (c *controller) fileScanned(meta m.FileMeta) {
	file := &m.File{
		FileMeta: m.FileMeta(meta),
		FileKind: m.FileRegular,
	}

	c.bySize[meta.Size] = append(c.bySize[meta.Size], file)

	archive := c.archives[meta.Root]
	archive.totalSize += meta.Size
	archive.byName[meta.FullName()] = file

	if file.Root == c.roots[0] {
		c.addToFolder(file, file.Size, file.ModTime)
	}
}

func (c *controller) addToFolder(file *m.File, size uint64, modTime time.Time) {
	parentFolder := c.folders[file.Path]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &m.File{
				FileMeta: m.FileMeta{
					FileId: m.FileId{
						Path: dir(file.Path),
						Name: name(file.Path),
					},
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				FileKind: m.FileFolder,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*m.File{file},
		}
		c.folders[file.Path] = parentFolder
	} else {
		sameName := []*m.File{}
		for _, entry := range parentFolder.entries {
			if file.Name == entry.Name {
				sameName = append(sameName, entry)
			}
		}
		if file.FileKind == m.FileFolder {
			for _, entry := range sameName {
				if entry.FileKind == m.FileFolder {
					return
				}
			}
		}
		for _, entry := range sameName {
			if file.FileKind == m.FileRegular &&
				entry.FileKind == m.FileRegular &&
				file.Size == entry.Size &&
				(entry.Hash == "" || file.Hash == entry.Hash) {

				entry.Hash = file.Hash
				return
			}
		}

		if len(sameName) > 0 {
			c.conflicts[file.FullName()] = struct{}{}
		}

		parentFolder.entries = append(parentFolder.entries, file)

		parentFolder.info.Size += size
		if parentFolder.info.ModTime.Before(modTime) {
			parentFolder.info.ModTime = modTime
		}
	}
	if file.Path != "" {
		c.addToFolder(parentFolder.info, size, modTime)
	}
}

func (c *controller) fileHashed(fileHash m.FileHashed) {
	archive := c.archives[fileHash.Root]
	file := archive.byName[fileHash.FullName()]
	file.Hash = fileHash.Hash
	c.byHash[fileHash.Hash] = append(c.byHash[fileHash.Hash], file)

	hashes := map[m.Hash]struct{}{}
	filesBySize := c.bySize[file.Size]
	for _, file := range filesBySize {
		hashes[file.Hash] = struct{}{}
	}

	if _, ok := hashes[""]; ok {
		return
	}

	for hash := range hashes {
		filesForHash := map[m.Root][]*m.File{}
		for _, file := range filesBySize {
			if file.Hash != hash {
				continue
			}
			filesForHash[file.Root] = append(filesForHash[file.Root], file)
		}

		originFiles := filesForHash[c.roots[0]]
		if len(originFiles) == 0 {
			c.addToFolder(file, file.Size, file.ModTime)
			c.hashStatus(hash, m.Absent)
			c.absentFiles++
		} else if len(originFiles) == 1 {
			for _, root := range c.roots {
				files := filesForHash[root]
				if len(files) != 1 || originFiles[0].FullName() != files[0].FullName() {
					c.hashStatus(hash, m.Pending)
					break
				}
			}
		} else {
			c.hashStatus(hash, m.Duplicate)
			c.duplicateFiles++
		}
	}
}

func (c *controller) makeSelectedVisible() {
	folder := c.folders[c.currentPath]
	if folder.offsetIdx > folder.selectedIdx {
		folder.offsetIdx = folder.selectedIdx
	}
	if folder.offsetIdx < folder.selectedIdx+1-c.fileTreeLines {
		folder.offsetIdx = folder.selectedIdx + 1 - c.fileTreeLines
	}
}

func (c *controller) filesHandled(handled m.FilesHandled) {
	log.Printf("filesHandled: ++++ %s", handled)
	c.hashStatus(handled.Hash, m.Status(m.Initial))
	c.pendingFiles--

	for _, fileId := range handled.Delete {
		c.removeFolderFile(m.FileId(fileId))
	}

renameBlock:
	for _, rename := range handled.Rename {

		c.removeFolderFile(rename.FileId)

		meta := c.archives[rename.Root].byName[rename.FullName()]
		meta.Path = rename.NewPath
		meta.Name = rename.NewName
		entries := c.folders[meta.Path].entries
		for _, entry := range entries {
			if meta.Name == entry.Name && meta.Hash == entry.Hash {
				continue renameBlock
			}
		}
		c.folders[meta.Path].entries = append(entries, meta)
	}

	if handled.Copy != nil {
		source := handled.Copy
		archive := c.archives[source.Root]
		meta := archive.byName[source.FullName()]
		c.totalCopied += meta.Size
		c.fileCopied = 0
		archive.progress.TotalHashed = 0
		if c.totalCopied == c.copySize {
			c.totalCopied = 0
			c.copySize = 0
			archive.progress.ProgressState = m.FileTreeHashed
		}
	}
}

func (c *controller) removeFolderFile(id m.FileId) {
	archive := c.archives[id.Root]
	meta := archive.byName[id.FullName()]
	path := id.Path
	entries := c.folders[path].entries
	for i, entry := range entries {
		if meta.Name == entry.Name && meta.Hash == entry.Hash {
			if i < len(entries)-1 {
				if c.folders[path].selected == entries[i] {
					c.folders[path].selected = entries[i+1]
				}
				entries[i] = entries[len(entries)-1]
				c.folders[path].entries = entries[:len(entries)-1]
			} else {
				if i > 0 && c.folders[path].selected == entries[i] {
					c.folders[path].selected = entries[i-1]
				} else {
					c.folders[path].selected = nil
				}
				c.folders[path].entries = entries[:len(entries)-1]
			}
			break
		}
	}
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
	for hash, files := range c.byHash {
		log.Printf("--- autoresolve hash %q", hash)
		for _, file := range files {
			log.Printf("    +++ autoresolve file %s", file)
			_, conflict := c.conflicts[file.FullName()]
			if file.Status == m.Pending && !conflict && file.Root == c.roots[0] {
				log.Printf("        ### autoresolved %s", file)
				c.keepFile(file)
			}
		}
	}
}

func (c *controller) fileCopyProgress(event m.FileCopyProgress) {
	c.fileCopied = uint64(event)
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
