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
	fileName := fileId.Name
	log.Printf("### keep %#v", fileId)

	keepFiles := map[m.Root]*w.File{}
	for _, entry := range c.files {
		if entry.Hash == file.Hash {
			root := entry.Root
			name := entry.NewId().Name
			if prevFile, ok := keepFiles[root]; ok {
				if name == fileName {
					keepFiles[root] = entry
				} else if name.Path == fileId.Path && name.Path != prevFile.Path {
					keepFiles[root] = entry
				} else if name.Base == fileId.Base && name.Base != prevFile.Base {
					keepFiles[root] = entry
				}
			} else {
				keepFiles[root] = entry
			}
		}
	}

	for _, entry := range c.files {
		if entry == file {
			continue
		}
		if entry.Hash == file.Hash {
			root := entry.Root
			keepFile := keepFiles[root]
			if entry == keepFile {
				if fileName != keepFile.Name {
					newId := m.Id{Root: keepFile.Root, Name: fileName}
					rename := c.ensureNameAvailable(newId)
					if rename != nil {
						cmd.Rename = append(cmd.Rename, *rename)
					}
					cmd.Rename = append(cmd.Rename, m.RenameFile{Id: keepFile.NewId(), NewName: fileName})
					log.Printf("+++ rename.1 %#v", m.RenameFile{Id: keepFile.NewId(), NewName: fileName})
					entry.Pending = true
					keepFile.PendingId = fileId
					c.pending[fileId] = keepFile
				}
			} else {
				cmd.Delete = append(cmd.Delete, entry.NewId())
				log.Printf("+++ delete.1 %#v", entry.NewId())
				entry.Pending = true
				entry.PendingId = m.Id{}
			}
		}
	}

	copy := m.CopyFile{From: fileId}
	for root, archive := range c.archives {
		if root == fileId.Root {
			continue
		}
		if _, ok := keepFiles[root]; !ok {
			newId := m.Id{Root: root, Name: fileName}
			rename := c.ensureNameAvailable(newId)
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
	for _, entry := range c.files {
		if entry.Hash == file.Hash && entry.Root != c.origin {
			cmd.Delete = append(cmd.Delete, entry.NewId())
		}
	}
	file.Pending = true
	log.Printf("deleteRegularFile: cmd: %s", cmd)
	c.archives[c.origin].scanner.Send(cmd)
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: implement
}
