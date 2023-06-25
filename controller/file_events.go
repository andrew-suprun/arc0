package controller

import (
	"arch/model"
	"log"
	"path/filepath"
	"time"
)

func (c *controller) archiveScanned(metas model.ArchiveScanned) {
	if len(metas) == 0 {
		return
	}
	for _, meta := range metas {
		c.fileScanned(meta)
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
	archive.byName[meta.Name] = file
}

func (c *controller) fileHashed(fileHash model.FileHashed) {
	archive := c.archives[fileHash.Root]
	file := archive.byName[fileHash.Name]
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

	filesForHash := map[string][]*model.File{}
	for _, file := range filesBySize {
		if file.Hash != fileHash.Hash {
			continue
		}
		filesForHash[file.Root] = append(filesForHash[file.Root], file)
	}

	uniqueNames := map[string]struct{}{}
	for _, files := range filesForHash {
		for _, file := range files {
			if _, exist := uniqueNames[file.Name]; !exist {
				uniqueNames[file.Name] = struct{}{}
				c.addToFolder(file, file.Size, file.ModTime)
			}
		}
	}

	originFiles := filesForHash[c.roots[0]]
	if len(originFiles) == 0 {
		c.hashStatus(fileHash.Hash, model.Absent)
	} else if len(originFiles) == 1 {
		for _, root := range c.roots {
			files := filesForHash[root]
			if len(files) != 1 || originFiles[0].Name != files[0].Name {
				c.keepFile(originFiles[0])
				break
			}
		}
	} else {
		c.hashStatus(fileHash.Hash, model.Duplicate)
	}
}

func (c *controller) addToFolder(file *model.File, size uint64, modTime time.Time) {
	parentFolder := c.folders[dir(file.Name)]
	if parentFolder == nil {
		parentFolder = &folder{
			info: &model.File{
				FileMeta: model.FileMeta{
					Name:    dir(file.Name),
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				Kind: model.FileFolder,
			},
			sortAscending: []bool{true, false, false, false},
			entries:       []*model.File{file},
		}
		c.folders[dir(file.Name)] = parentFolder
	} else {
		folderAlreadyExists := false
		switch file.Kind {
		case model.FileRegular:
			parentFolder.entries = append(parentFolder.entries, file)
		case model.FileFolder:
			for _, entry := range parentFolder.entries {
				if entry.Kind == model.FileFolder && name(file.Name) == name(entry.Name) {
					folderAlreadyExists = true
					break
				}
			}
			if !folderAlreadyExists {
				parentFolder.entries = append(parentFolder.entries, file)
			}
		}
		parentFolder.info.Size += size
		if parentFolder.info.ModTime.Before(modTime) {
			parentFolder.info.ModTime = modTime
		}
	}
	if dir(file.Name) != "" {
		c.addToFolder(parentFolder.info, size, modTime)
	}
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

func (c *controller) makeSelectedVisible() {
	folder := c.folders[c.currentPath]
	if folder.selected == nil {
		return
	}

	idx := -1
	for i := range folder.entries {
		if folder.selected == folder.entries[i] {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if folder.lineOffset > idx {
			folder.lineOffset = idx
		}
		if folder.lineOffset < idx+1-c.fileTreeLines {
			folder.lineOffset = idx + 1 - c.fileTreeLines
		}
	}
}

func (c *controller) filesHandled(handled model.FilesHandled) {
	log.Printf("### files handled for %s", handled.String())

	c.hashStatus(handled.Hash, model.Resolved)

	if handled.Copy != nil {
		source := handled.Copy
		archive := c.archives[source.SourceRoot]
		meta := archive.byName[source.Name]
		c.totalCopied += meta.Size
		archive.progress.Processed = 0
		if c.totalCopied == c.copySize {
			c.totalCopied = 0
			c.copySize = 0
			archive.progress.ProgressState = model.HashingFileTreeComplete
		}
	}
}

func (c *controller) progressEvent(event model.Progress) {
	c.archives[event.Root].progress = event

	if event.ProgressState == model.HashingFileTreeComplete {
		for _, archive := range c.archives {
			if archive.progress.ProgressState != model.HashingFileTreeComplete {
				return
			}
		}
		c.fileHandler = c.fs.NewFileHandler()
		for _, msg := range c.messages {
			c.fileHandler.Send(msg)
		}
	}
}
