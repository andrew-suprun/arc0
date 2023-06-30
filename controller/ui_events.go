package controller

import (
	"arch/model"
	"log"
	"os/exec"
	"sort"
	"time"
)

func (c *controller) mouseTarget(cmd any) {
	folder := c.folders[c.currentPath]
	switch cmd := cmd.(type) {
	case selectFile:
		if folder.selected == cmd && time.Since(c.lastMouseEventTime).Seconds() < 0.5 {
			c.enter()
		} else {
			for idx, entry := range folder.entries {
				if entry == cmd {
					folder.selectedIdx = idx
					folder.selected = nil
					break
				}
			}
		}
		c.lastMouseEventTime = time.Now()

	case selectFolder:
		c.currentPath = cmd.FileId.FullName()

	case sortColumn:
		if cmd == folder.sortColumn {
			folder.sortAscending[folder.sortColumn] = !folder.sortAscending[folder.sortColumn]
		} else {
			folder.sortColumn = cmd
		}
	}
}

func (c *controller) selectFirst() {
	folder := c.folders[c.currentPath]
	if len(folder.entries) > 0 {
		folder.selected = folder.entries[0]
	}
}

func (c *controller) selectLast() {
	folder := c.folders[c.currentPath]
	entries := folder.entries
	if len(entries) > 0 {
		folder.selected = entries[len(entries)-1]
	}
}

func (c *controller) moveSelection(lines int) {
	folder := c.folders[c.currentPath]
	folder.selectedIdx += lines
	if folder.selectedIdx < 0 {
		folder.selectedIdx = 0
	} else if folder.selectedIdx >= len(folder.entries) {
		folder.selectedIdx = len(folder.entries) - 1
	}
	folder.selected = nil
}

func (c *controller) enter() {
	folder := c.folders[c.currentPath]
	selected := folder.selected
	if selected != nil {
		if selected.Kind == model.FileFolder {
			c.currentPath = selected.FullName()
		} else {
			exec.Command("open", selected.AbsName()).Start()
		}
	}
}

func (c *controller) shiftOffset(lines int) {
	folder := c.folders[c.currentPath]
	nEntries := len(folder.entries)
	folder.offsetIdx += lines
	if folder.offsetIdx < 0 {
		folder.offsetIdx = 0
	} else if folder.offsetIdx >= nEntries {
		folder.offsetIdx = nEntries - 1
	}
}

func (c *controller) keepSelected() {
	c.keepFile(c.folders[c.currentPath].selected)
}

func (c *controller) keepFile(file *model.File) {
	if file == nil || file.Kind != model.FileRegular {
		return
	}
	log.Printf("keepFile: file %s", file)
	msg := model.HandleFiles{Hash: file.Hash}

	filesForHash := c.byHash[file.Hash]
	byArch := map[string][]*model.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}

	copyFiles := &model.CopyFile{FileId: file.FileId}

	for _, root := range c.roots {
		if len(byArch[root]) == 0 {
			copyFiles.TargetRoots = append(copyFiles.TargetRoots, root)
		}
	}
	if len(copyFiles.TargetRoots) > 0 {
		msg.Copy = copyFiles
		c.copySize += file.Size
	}

	for _, root := range c.roots {
		archFiles := byArch[root]

		keepIdx := 0
		for i, archFile := range archFiles {
			if archFile == file || archFile.FullName() == file.FullName() {
				keepIdx = i
				break
			}
		}
		for i, archFile := range archFiles {
			if i == keepIdx {
				if file.FullName() != archFile.FullName() {
					msg.Rename = append(msg.Rename, model.RenameFile{
						FileId: model.FileId{
							Root: root,
							Path: archFile.Path,
							Name: archFile.Name,
						},
						NewPath: file.Path,
						NewBase: file.Name,
					})
				}
			} else {
				msg.Delete = append(msg.Delete, model.DeleteFile{
					Root: archFile.Root,
					Path: archFile.Path,
					Name: archFile.Name,
				})
			}
		}
	}

	log.Printf("keepFile: msg: %s", msg)
	if msg.Copy != nil || msg.Rename != nil || len(msg.Delete) > 0 {
		for _, file := range filesForHash {
			c.updateFolderStatus(file.Path)
		}
		c.pendingFiles++
		c.fs.Send(msg)
	}
}

func (c *controller) tab() {
	log.Printf("### tab")
	selected := c.folders[c.currentPath].selected
	if selected == nil || selected.Kind != model.FileRegular {
		return
	}
	name := selected.FullName()
	hash := selected.Hash

	byHash := c.byHash[hash]
	uniqueNames := map[string]struct{}{}
	for _, meta := range byHash {
		uniqueNames[meta.FullName()] = struct{}{}
	}
	names := []string{}
	for name := range uniqueNames {
		names = append(names, name)
	}
	sort.Strings(names)
	idx := 0
	for ; idx < len(names); idx++ {
		if name == names[idx] {
			break
		}
	}
	name = names[(idx+1)%len(names)]
	c.currentPath = dir(name)
	folder := c.folders[c.currentPath]
	for _, meta := range folder.entries {
		if name == meta.FullName() && hash == meta.Hash {
			folder.selected = meta
			break
		}
	}
	c.makeSelectedVisible()
}

func (c *controller) updateFolderStatus(path string) {
	currentFolder := c.folders[path]
	status := currentFolder.info.Status
	currentFolder.info.Status = model.Resolved
	for _, entry := range currentFolder.entries {
		currentFolder.info.MergeStatus(entry)
	}
	if path != "" && currentFolder.info.Status != status {
		c.updateFolderStatus(dir(path))
	}
}

func (c *controller) deleteSelected() {
	c.deleteFile(c.folders[c.currentPath].selected)
}

func (c *controller) deleteFile(file *model.File) {
	if file == nil {
		return
	}
	status := file.Status
	if status != model.Absent {
		return
	}

	c.absentFiles--
	if file.Kind == model.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file)
	}
	c.updateFolderStatus(file.Path)
}

func (c *controller) deleteRegularFile(file *model.File) {
	c.hashStatus(file.Hash, model.ResolveAbsent)

	filesForHash := c.byHash[file.Hash]
	byArch := map[string][]*model.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}
	if len(byArch[c.roots[0]]) > 0 {
		return
	}

	msg := model.HandleFiles{Hash: file.Hash}
	for _, file := range filesForHash {
		msg.Delete = append(msg.Delete, model.DeleteFile{
			Root: file.Root,
			Path: file.Path,
			Name: file.Name,
		})
	}
	c.pendingFiles++
	c.fs.Send(msg)
}

func (c *controller) deleteFolderFile(file *model.File) {
	folder := c.folders[file.FullName()]
	for _, entry := range folder.entries {
		c.deleteFile(entry)
	}
}
