package controller

import (
	m "arch/model"
	"log"
	"strings"
)

func (c *controller) keepFile(file *m.File) {
	log.Printf("keepFile: file: %s", file)

	if file == nil {
		log.Panic("keepFile(nil)")
	}
	if file.State <= m.Pending {
		return
	}

	scanner := c.archives[c.origin].scanner
	entries := c.byHash[file.Hash]
	pending := false

	fileName := file.Name
	keepFiles := map[m.Root]*m.File{}
	for _, entry := range entries {
		root := entry.Root
		name := entry.Name
		if prevFile, ok := keepFiles[root]; ok {
			if name == fileName {
				keepFiles[root] = entry
			} else if name.Path == fileName.Path && name.Path != prevFile.Path {
				keepFiles[root] = entry
			} else if name.Base == fileName.Base && name.Base != prevFile.Base {
				keepFiles[root] = entry
			}
		} else {
			keepFiles[root] = entry
		}
	}

	for _, entry := range entries {
		if entry.Id == file.Id {
			continue
		}
		root := entry.Root
		keepFile := keepFiles[root]
		if entry == keepFile {
			if fileName != keepFile.Name {
				newId := m.Id{Root: keepFile.Root, Name: fileName}
				scanner.Send(m.RenameFile{From: keepFile.Id, To: newId, Hash: file.Hash})
				pending = true
				keepFile.State = m.Pending
				delete(c.byId, keepFile.Name)
				c.byId[fileName] = keepFile
				c.removeEntry(keepFile.Id)
				c.addFile(keepFile)
				keepFile.Id = newId
			}
		} else {
			scanner.Send(m.DeleteFile{Id: entry.Id, Hash: file.Hash})
			pending = true
			delete(c.byId, file.Name)
			c.removeEntry(file.Id)
		}
	}

	copy := m.CopyFile{From: file.Id, Hash: file.Hash}
	for _, root := range c.roots {
		if root == file.Root {
			continue
		}
		if _, ok := keepFiles[root]; !ok {
			newId := m.Id{Root: root, Name: fileName}
			copy.To = append(copy.To, newId)
		}
	}
	if len(copy.To) > 0 {
		scanner.Send(copy)
		pending = true
		file.State = m.Pending
		c.copySize += file.Size
	}
	if pending {
		if file.State == m.Duplicate {
			c.duplicateFiles--
		} else if file.State == m.Absent {
			c.absentFiles--
		}
		c.pendingFiles++
		c.setState(file.Hash, m.Pending)
	}
}

func (c *controller) deleteFile(file *m.File) {
	if file.State != m.Absent {
		return
	}
	if file.Kind == m.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file.Hash)
	}

	c.setState(file.Hash, m.Pending)
}

func (c *controller) deleteRegularFile(hash m.Hash) {
	c.setState(hash, m.Pending)
	c.absentFiles--
	c.pendingFiles++
	for _, entry := range c.byHash[hash] {
		c.archives[c.origin].scanner.Send(m.DeleteFile{Id: entry.Id, Hash: entry.Hash})
		delete(c.byId, entry.Name)
		c.removeEntry(entry.Id)
	}
}

func (c *controller) deleteFolderFile(file *m.File) {
	path := file.Name.String()
	hashes := map[m.Hash]struct{}{}

	for _, entry := range c.byId {
		if entry.State == m.Absent && strings.HasPrefix(entry.Path.String(), path) {
			hashes[entry.Hash] = struct{}{}
		}
	}

	for hash := range hashes {
		c.deleteRegularFile(hash)
	}
}
