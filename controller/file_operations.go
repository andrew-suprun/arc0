package controller

import (
	m "arch/model"
	w "arch/widgets"
	"strings"
)

func (c *controller) keepFile(file *m.File) {
	if file == nil || !c.archivesScanned || c.state[file.Hash] == w.Pending {
		return
	}

	scanner := c.archives[c.origin].scanner
	c.state[file.Hash] = w.Pending
	files := c.files[file.Hash]

	fileName := file.Name
	keepFiles := map[m.Root]*m.File{}
	for _, entry := range files {
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

	for _, entry := range files {
		if entry.Id == file.Id {
			continue
		}
		root := entry.Root
		keepFile := keepFiles[root]
		if entry == keepFile {
			if fileName != keepFile.Name {
				newId := m.Id{Root: keepFile.Root, Name: fileName}
				scanner.Send(m.RenameFile{From: keepFile.Id, To: newId, Hash: file.Hash})
				keepFile.Id = newId
			}
		} else {
			scanner.Send(m.DeleteFile{Id: entry.Id, Hash: file.Hash})
			for i, file := range files {
				if file.Id == entry.Id {
					files[i] = files[len(files)-1]
					c.files[file.Hash] = files[:len(files)-1]
					break
				}
			}
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
			newFile := &m.File{
				Id:      newId,
				Size:    file.Size,
				ModTime: file.ModTime,
				Hash:    file.Hash,
			}
			c.files[file.Hash] = append(c.files[file.Hash], newFile)
		}
	}
	if len(copy.To) > 0 {
		scanner.Send(copy)
		c.copySize += file.Size
	}
}

func (c *controller) deleteFile(file *w.File) {
	if file.Kind == w.FileFolder {
		c.deleteFolderFile(file)
	} else {
		c.deleteRegularFile(file.Hash)
	}

	c.state[file.Hash] = w.Pending
}

func (c *controller) deleteRegularFile(hash m.Hash) {
	if c.state[hash] != w.Absent {
		return
	}
	c.state[hash] = w.Pending
	c.every(func(entry *m.File) {
		if entry.Hash == hash {
			c.archives[c.origin].scanner.Send(m.DeleteFile{Id: entry.Id, Hash: entry.Hash})
			files := c.files[hash]
			for i, file := range files {
				if file.Id == entry.Id {
					files[i] = files[len(files)-1]
					c.files[file.Hash] = files[:len(files)-1]
					break
				}
			}
		}
	})
}

func (c *controller) deleteFolderFile(file *w.File) {
	path := file.Name.String()
	hashes := map[m.Hash]struct{}{}
	c.every(func(entry *m.File) {
		if c.state[entry.Hash] == w.Absent && strings.HasPrefix(entry.Path.String(), path) {
			hashes[entry.Hash] = struct{}{}
		}
	})

	for hash := range hashes {
		c.deleteRegularFile(hash)
	}
}
