package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"strings"
)

func (c *controller) keepFile(file *w.File) {
	if file == nil || file.Kind != w.FileRegular || !c.archivesScanned || c.state[file.Hash] == w.Pending {
		return
	}

	c.state[file.Hash] = w.Pending

	cmd := m.HandleFiles{Hash: file.Hash}

	fileName := file.Name
	keepFiles := map[m.Root]*m.File{}
	c.do(func(entry *m.File) bool {
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
		return true
	})
	c.do(func(entry *m.File) bool {
		if entry.Id == file.Id {
			return true
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
					cmd.Rename = append(cmd.Rename, m.RenameFile{Id: keepFile.Id, NewId: newId})
				}
			} else {
				cmd.Delete = append(cmd.Delete, entry.Id)
			}
		}
		return true
	})

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
				File: file.File,
				Kind: file.Kind,
			}
			newFile.Id = newId
		}
	}
	if len(copy.To) > 0 {
		cmd.Copy = &copy
		c.copySize += file.Size
	}
	if len(cmd.Delete) > 0 || len(cmd.Rename) > 0 || cmd.Copy != nil {
		c.archives[c.origin].scanner.Send(cmd)
		c.state[file.Hash] = w.Pending
	}
}

func (c *controller) deleteFile(file *w.File) {
	if file == nil || !c.archivesScanned {
		return
	}

	c.state[file.Hash] = w.Pending

	if file.Root == "" {
		c.deleteFolderFile(file)
	} else if c.state[file.Hash] == w.Absent {
		c.deleteRegularFile(file.Hash)
	}
}

func (c *controller) deleteRegularFile(hash m.Hash) {
	cmd := m.HandleFiles{Hash: hash}
	c.do(func(entry *m.File) bool {
		if entry.Hash == hash && c.state[entry.Hash] == w.Absent {
			cmd.Delete = append(cmd.Delete, entry.Id)
		}
		return true
	})
	c.archives[c.origin].scanner.Send(cmd)
}

func (c *controller) deleteFolderFile(file *w.File) {
	path := file.Name.String()
	hashes := map[m.Hash]struct{}{}
	c.do(func(entry *m.File) bool {
		if c.state[entry.Hash] == w.Absent && strings.HasPrefix(entry.Path.String(), path) {
			hashes[entry.Hash] = struct{}{}
		}
		return true
	})

	for hash := range hashes {
		c.deleteRegularFile(hash)
	}
}

func (c *controller) ensureNameAvailable(id m.Id) *m.RenameFile {
	file := c.find(func(entry *m.File) bool {
		return entry.Id == id
	})
	if file != nil {
		newId := c.newName(id)
		file.Id = newId
		return &m.RenameFile{Id: id, NewId: newId}
	}
	return nil
}

func (c *controller) newName(id m.Id) m.Id {
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
		file := c.find(func(entry *m.File) bool {
			return id.Path == entry.Path && newName == entry.Base.String()
		})

		if file == nil {
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
