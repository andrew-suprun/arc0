package controller

import (
	m "arch/model"
	w "arch/widgets"
	"log"
)

func (c *controller) keepFile(file *w.File) {
	if file == nil || file.FileKind != w.FileRegular || file.Hash == "" || file.Pending {
		return
	}

	cmd := m.HandleFiles{Hash: file.Hash}

	fileId := file.NewId()
	fileName := fileId.FullName()
	log.Printf("### keep %#v", fileId)

	keepFiles := map[m.Root]*w.File{}
	for root, archive := range c.archives {
		for _, entry := range archive.files {
			if entry.Hash == file.Hash {
				name := entry.NewName()
				if prevFile, ok := keepFiles[root]; ok {
					if name == fileId.FullName() {
						keepFiles[root] = entry
					} else if name.Path == fileId.Path && name.Path != prevFile.Path {
						keepFiles[root] = entry
					} else if name.Name == fileId.Name && name.Name != prevFile.Name {
						keepFiles[root] = entry
					}
				} else {
					keepFiles[root] = entry
				}
			}
		}
	}

	for root, archive := range c.archives {
		for _, entry := range archive.files {
			if entry == file {
				continue
			}
			if entry.Hash == file.Hash {
				keepFile := keepFiles[root]
				if entry == keepFile {
					if fileName != keepFile.FullName() {
						newId := m.FileId{Root: keepFile.Root, Path: fileId.Path, Name: fileId.Name}
						rename := archive.ensureNameAvailable(newId)
						if rename != nil {
							cmd.Rename = append(cmd.Rename, *rename)
						}
						cmd.Rename = append(cmd.Rename, m.RenameFile{FileId: keepFile.NewId(), NewFullName: fileName})
						log.Printf("+++ rename.1 %#v", m.RenameFile{FileId: keepFile.NewId(), NewFullName: fileName})
						entry.Pending = true
						keepFile.PendingName = fileName
						archive.pending[fileName] = keepFile
					}
				} else {
					cmd.Delete = append(cmd.Delete, entry.NewId())
					log.Printf("+++ delete.1 %#v", entry.NewId())
					entry.Pending = true
					entry.PendingName = m.FullName{}
				}
			}
		}
	}

	copy := m.CopyFile{From: fileId}
	for root, archive := range c.archives {
		if root == fileId.Root {
			continue
		}
		if _, ok := keepFiles[root]; !ok {
			newId := m.FileId{Root: root, Path: fileName.Path, Name: fileName.Name}
			rename := archive.ensureNameAvailable(newId)
			if rename != nil {
				cmd.Rename = append(cmd.Rename, *rename)
			}
			copy.To = append(copy.To, root)
			log.Printf("+++ copy %q to %q", fileId, root)
			archive.copySize += file.Size
		}
	}
	if len(copy.To) > 0 {
		cmd.Copy = &copy
	}
	if len(cmd.Delete) > 0 || len(cmd.Rename) > 0 || cmd.Copy != nil {
		c.archives[c.origin].scanner.Send(cmd)
		file.Pending = true
	}
}

func (c *controller) deleteFile(file *w.File) {
	if file == nil || file.Hash == "" || file.Pending || c.presence[file.Hash] != w.Absent {
		return
	}

	if file.FileKind == w.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file)
	}
}

func (c *controller) deleteRegularFile(file *w.File) {
	cmd := m.HandleFiles{Hash: file.Hash}
	for _, root := range c.roots[1:] {
		archive := c.archives[root]
		for _, entry := range archive.files {
			if entry.Hash == file.Hash {
				cmd.Delete = append(cmd.Delete, entry.NewId())
			}
		}
	}
	file.Pending = true
	log.Printf("deleteRegularFile: cmd: %s", cmd)
	c.archives[c.origin].scanner.Send(cmd)
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: implement
}
