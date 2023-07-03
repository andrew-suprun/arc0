package controller

import (
	m "arch/model"
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
		c.currentPath = m.Path(cmd.FileId.FullName().String())

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
	}
	if folder.selectedIdx >= len(folder.entries) {
		folder.selectedIdx = len(folder.entries) - 1
	}
	folder.selected = nil
}

func (c *controller) enter() {
	folder := c.folders[c.currentPath]
	selected := folder.selected
	if selected != nil {
		if selected.FileKind == m.FileFolder {
			c.currentPath = m.Path(selected.FullName().String())
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

func (c *controller) keepFile(file *m.File) {
	if file == nil || file.FileKind != m.FileRegular {
		return
	}
	log.Printf("keepFile: file %s", file)
	msg := m.HandleFiles{Hash: file.Hash}

	filesForHash := c.byHash[file.Hash]
	byArch := map[m.Root][]*m.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}

	copyFiles := &m.CopyFile{FileId: file.FileId}

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
					msg.Rename = append(msg.Rename, m.RenameFile{
						FileId: m.FileId{
							Root: root,
							Path: archFile.Path,
							Name: archFile.Name,
						},
						NewPath: file.Path,
						NewName: file.Name,
					})
				}
			} else {
				msg.Delete = append(msg.Delete, m.DeleteFile{
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
	selected := c.folders[c.currentPath].selected
	if selected == nil || selected.FileKind != m.FileRegular || selected.Status != m.Duplicate {
		return
	}
	name := selected.FullName().String()
	hash := selected.Hash
	log.Printf("### tab: name=%q hash=%q", name, hash)

	byHash := c.byHash[hash]
	uniqueNames := map[m.FullName]struct{}{}
	for _, meta := range byHash {
		if meta.Root == c.roots[0] {
			log.Printf("### tab: name=%q hash=%q", meta.FullName(), hash)
			uniqueNames[meta.FullName()] = struct{}{}
		}
	}
	names := []string{}
	for name := range uniqueNames {
		names = append(names, name.String())
	}
	sort.Strings(names)
	idx := 0
	for ; idx < len(names); idx++ {
		if name == names[idx] {
			break
		}
	}
	name = names[(idx+1)%len(names)]
	c.currentPath = dir(m.Path(name))
	folder := c.folders[c.currentPath]
	for _, meta := range folder.entries {
		if name == meta.FullName().String() && hash == meta.Hash {
			folder.selected = meta
			break
		}
	}
	c.makeSelectedVisible()
}

func (c *controller) updateFolderStatus(path m.Path) {
	log.Printf("### updateFolderStatus path=%q", path)
	currentFolder := c.folders[path]
	status := currentFolder.info.Status
	currentFolder.info.Status = m.Resolved
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

func (c *controller) deleteFile(file *m.File) {
	if file == nil {
		return
	}
	status := file.Status
	if status != m.Absent {
		return
	}

	c.absentFiles--
	if file.FileKind == m.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file)
	}
	c.updateFolderStatus(file.Path)
}

func (c *controller) deleteRegularFile(file *m.File) {
	c.hashStatus(file.Hash, m.Pending)

	filesForHash := c.byHash[file.Hash]
	byArch := map[m.Root][]*m.File{}
	for _, fileForHash := range filesForHash {
		byArch[fileForHash.Root] = append(byArch[fileForHash.Root], fileForHash)
	}
	if len(byArch[c.roots[0]]) > 0 {
		return
	}

	msg := m.HandleFiles{Hash: file.Hash}
	for _, file := range filesForHash {
		msg.Delete = append(msg.Delete, m.DeleteFile{
			Root: file.Root,
			Path: file.Path,
			Name: file.Name,
		})
	}
	c.pendingFiles++
	c.fs.Send(msg)
}

func (c *controller) deleteFolderFile(file *m.File) {
	folder := c.folders[m.Path(file.FullName().String())]
	for _, entry := range folder.entries {
		c.deleteFile(entry)
	}
}
