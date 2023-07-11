package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"log"
	"strings"
)

func (c *controller) keepFile(file *w.File) {
	if file == nil || file.FileKind != w.FileRegular || file.Hash == "" || c.state[file.Hash] == w.Pending {
		return
	}
	c.state[file.Hash] = w.Pending

	cmd := m.HandleFiles{Hash: file.Hash}

	fileName := file.Name
	keepFiles := map[m.Root]*w.File{}
	for _, entry := range c.files {
		if entry.Hash == file.Hash {
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
					cmd.Rename = append(cmd.Rename, m.RenameFile{Id: keepFile.Id, NewName: fileName})
					c.files[newId] = keepFile
					delete(c.files, keepFile.Id)
					keepFile.Id = newId
				}
			} else {
				cmd.Delete = append(cmd.Delete, entry.Id)
				delete(c.files, entry.Id)
			}
		}
	}

	copy := m.CopyFile{From: file.Id}
	for _, root := range c.roots {
		if root == file.Root {
			continue
		}
		if _, ok := keepFiles[root]; !ok {
			newId := m.Id{Root: root, Name: fileName}
			rename := c.ensureNameAvailable(newId)
			if rename != nil {
				cmd.Rename = append(cmd.Rename, *rename)
			}
			copy.To = append(copy.To, root)
			newFile := &w.File{
				FileMeta: file.FileMeta,
				FileKind: file.FileKind,
				Hash:     file.Hash,
			}
			newFile.Id = newId
			c.files[newId] = newFile
		}
	}
	if len(copy.To) > 0 {
		cmd.Copy = &copy
		c.copySize += file.Size
		log.Printf("keepFile: copySize: %d", c.copySize)
	}
	if len(cmd.Delete) > 0 || len(cmd.Rename) > 0 || cmd.Copy != nil {
		c.archives[c.origin].scanner.Send(cmd)
	}
}

func (c *controller) deleteFile(file *w.File) {
	if file == nil || file.Hash == "" || c.state[file.Hash] != w.Absent {
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
			cmd.Delete = append(cmd.Delete, entry.Id)
			delete(c.files, entry.Id)
		}
	}
	c.archives[c.origin].scanner.Send(cmd)
}

func (c *controller) deleteFolderFile(file *w.File) {
	// TODO: implement
}

func (c *controller) ensureNameAvailable(id m.Id) *m.RenameFile {
	file := c.files[id]
	if file != nil {
		newId := c.newName(id)
		c.files[newId] = file
		delete(c.files, id)
		file.Id = newId
		return &m.RenameFile{Id: id, NewName: newId.Name}
	}
	return nil
}

func (a *controller) newName(id m.Id) m.Id {
	parts := strings.Split(id.Base.String(), ".")

	var part string
	if len(parts) == 1 {
		part = stripIdx(parts[0])
	} else {
		part = stripIdx(parts[len(parts)-2])
	}
	for idx := 1; ; idx++ {
		var newName string
		if len(parts) == 1 {
			newName = fmt.Sprintf("%s [%d]", part, idx)
		} else {
			parts[len(parts)-2] = fmt.Sprintf("%s [%d]", part, idx)
			newName = strings.Join(parts, ".")
		}
		exists := false
		for _, entity := range a.files {
			if id.Path == entity.Path && newName == entity.Base.String() {
				exists = true
				break
			}
		}
		if !exists {
			return m.Id{
				Root: id.Root,
				Name: m.Name{Path: id.Path, Base: m.Base(newName)},
			}
		}
	}
}

type stripIdxState int

const (
	expectCloseBracket stripIdxState = iota
	expectDigit
	expectDigitOrOpenBracket
	expectOpenBracket
	expectSpace
	done
)

func stripIdx(name string) string {
	state := expectCloseBracket
	i := len(name) - 1
	for ; i >= 0; i-- {
		ch := name[i]
		if ch == ']' && state == expectCloseBracket {
			state = expectDigit
		} else if ch >= '0' && ch <= '9' && (state == expectDigit || state == expectDigitOrOpenBracket) {
			state = expectDigitOrOpenBracket
		} else if ch == '[' && state == expectDigitOrOpenBracket {
			state = expectSpace
		} else if ch == ' ' && state == expectSpace {
			break
		} else {
			return name
		}
	}
	return name[:i]
}
