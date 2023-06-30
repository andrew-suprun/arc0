package controller

import (
	"arch/model"
	"log"
	"path/filepath"
	"time"
)

func (c *controller) archiveScanned(tree model.ArchiveScanned) {
	for _, meta := range tree.Metas {
		c.fileScanned(meta)
	}
	c.archives[tree.Root].progress.ProgressState = model.FileTreeScanned
	for _, archive := range c.archives {
		if archive.progress.ProgressState != model.FileTreeScanned {
			return
		}
	}
	for _, archive := range c.archives {
		archive.scanner.HashArchive()
	}
}

func (c *controller) fileScanned(meta model.FileMeta) {
	file := &model.File{
		FileMeta: model.FileMeta(meta),
		Kind:     model.FileRegular,
	}

	c.bySize[meta.Size] = append(c.bySize[meta.Size], file)

	archive := c.archives[meta.Root]
	archive.totalSize += meta.Size
	archive.byName[meta.FullName()] = file

	if file.Root == c.roots[0] {
		c.addToFolder(file, file.Size, file.ModTime)
	}
}

func (c *controller) addToFolder(file *model.File, size uint64, modTime time.Time) {
	parentFolder := c.folders[file.Path]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &model.File{
				FileMeta: model.FileMeta{
					FileId: model.FileId{
						Path: dir(file.Path),
						Name: name(file.Path),
					},
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				Kind: model.FileFolder,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*model.File{file},
		}
		c.folders[file.Path] = parentFolder
	} else {
		sameName := []*model.File{}
		for _, entry := range parentFolder.entries {
			if file.Name == entry.Name {
				sameName = append(sameName, entry)
			}
		}
		if file.Kind == model.FileFolder {
			for _, entry := range sameName {
				if entry.Kind == model.FileFolder {
					return
				}
			}
		}
		for _, entry := range sameName {
			if file.Kind == model.FileRegular &&
				entry.Kind == model.FileRegular &&
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

func (c *controller) fileHashed(fileHash model.FileHashed) {
	archive := c.archives[fileHash.Root]
	file := archive.byName[fileHash.FullName()]
	file.Hash = fileHash.Hash
	c.byHash[fileHash.Hash] = append(c.byHash[fileHash.Hash], file)

	hashes := map[string]struct{}{}
	filesBySize := c.bySize[file.Size]
	for _, file := range filesBySize {
		hashes[file.Hash] = struct{}{}
	}

	if _, ok := hashes[""]; ok {
		return
	}

	for hash := range hashes {
		filesForHash := map[string][]*model.File{}
		for _, file := range filesBySize {
			if file.Hash != hash {
				continue
			}
			filesForHash[file.Root] = append(filesForHash[file.Root], file)
		}

		originFiles := filesForHash[c.roots[0]]
		if len(originFiles) == 0 {
			c.addToFolder(file, file.Size, file.ModTime)
			c.hashStatus(hash, model.Absent)
			c.absentFiles++
		} else if len(originFiles) == 1 {
			for _, root := range c.roots {
				files := filesForHash[root]
				if len(files) != 1 || originFiles[0].FullName() != files[0].FullName() {
					c.hashStatus(hash, model.AutoResolve)
					break
				}
			}
		} else {
			c.hashStatus(hash, model.Duplicate)
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

func (c *controller) filesHandled(handled model.FilesHandled) {
	log.Printf("filesHandled: ++++ %s", handled)
	c.hashStatus(handled.Hash, model.ResulutionStatus(model.Initial))
	c.pendingFiles--

	for _, fileId := range handled.Delete {
		c.removeFolderFile(model.FileId(fileId))
	}

renameBlock:
	for _, rename := range handled.Rename {

		c.removeFolderFile(rename.FileId)

		meta := c.archives[rename.Root].byName[rename.FullName()]
		meta.Path = rename.NewPath
		meta.Name = rename.NewBase
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
			archive.progress.ProgressState = model.FileTreeHashed
		}
	}
}

func (c *controller) removeFolderFile(id model.FileId) {
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

func (c *controller) scanProgress(event model.ScanProgress) {
	c.archives[event.Root].progress = event

	if event.ProgressState == model.FileTreeHashed {
		for _, archive := range c.archives {
			if archive.progress.ProgressState != model.FileTreeHashed {
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
			if file.Status == model.AutoResolve && !conflict && file.Root == c.roots[0] {
				log.Printf("        ### autoresolved %s", file)
				c.keepFile(file)
			}
		}
	}
}

func (c *controller) fileCopyProgress(event model.FileCopyProgress) {
	c.fileCopied = uint64(event)
}

func dir(path string) string {
	path = filepath.Dir(path)
	if path == "." {
		return ""
	}
	return path
}

func name(path string) string {
	return filepath.Base(path)
}
