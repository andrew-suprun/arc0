package controller

import (
	m "arch/model"
	"fmt"
	"log"
	"strings"
)

func (c *controller) archiveScanned(event m.ArchiveScanned) {
	c.addFiles(event)

	archive := c.archives[event.Root]
	for _, file := range event.Files {
		archive.totalSize += file.Size
	}
	archive.progressState = m.ProgressScanned

	allScanned := true
	for _, archive := range c.archives {
		if archive.progressState != m.ProgressScanned {
			allScanned = false
			break
		}
	}

	if allScanned {
		for _, archive := range c.archives {
			archive.scanner.Send(m.HashArchive{})
		}
	}
}

func (c *controller) fileHashed(event m.FileHashed) {
	log.Printf("file hashed: %s", event)
	file := c.byId[event.Id]
	file.Hash = event.Hash

	archive := c.archives[event.Root]
	archive.totalHashed += file.Size
	archive.fileHashed = 0

	hashes := map[m.Hash]struct{}{}
	files := c.bySize[file.Size]
	for _, file := range files {
		log.Printf("file hashed:     file: %s", file)
		if file.Hash == "" {
			log.Printf("file hashed:     skipped")
			return
		}
		hashes[file.Hash] = struct{}{}
	}
	for hash := range hashes {
		log.Printf("file hashed:     hash: %q", hash)
		var entries []*m.File
		var origins []*m.File
		names := map[m.Name]struct{}{}
		for _, entry := range files {
			if entry.Hash == hash {
				entries = append(entries, entry)
				names[entry.Name] = struct{}{}
				if entry.Root == c.origin {
					origins = append(origins, entry)
					log.Printf("file hashed:         origin: %s", entry)
				}
			}
		}
		switch len(origins) {
		case 0:
			for _, entry := range entries {
				entry.State = m.Absent
				c.addCopyFile(entry)
			}

		case 1:
			keep := origins[0]
			keep.State = m.Autoresolve
			c.keepFile(keep)

		default:
			for _, entry := range entries {
				entry.State = m.Duplicate
			}
		}
	}
}

func (c *controller) addFiles(event m.ArchiveScanned) {
	for _, file := range event.Files {
		entry := &m.File{
			Meta:  file,
			Kind:  m.FileRegular,
			State: m.Initial,
		}
		c.byId[entry.Id] = entry

		entries := c.bySize[file.Size]
		entries = append(entries, entry)
		c.bySize[file.Size] = entries

		if event.Root == c.origin {
			c.addFile(entry)
		}
	}
	c.setInitialSelection("")
	c.currentPath = ""
}

func (c *controller) addFile(file *m.File) {
	folder := c.getFolder(file.Path)
	folder.entries[file.Base] = file
	name := file.ParentName()
	for name.Base != "." {
		parentFolder := c.getFolder(name.Path)
		item := parentFolder.entries[name.Base]
		if item != nil {
			if item.Kind != m.FileFolder {
				log.Panicf("ERROR: Name collision in controller.addEntry(): %q", name.Base)
			}
			item.Size += file.Size
			if item.ModTime.Before(file.ModTime) {
				item.ModTime = file.ModTime
			}
		} else {
			folderEntry := &m.File{
				Meta: m.Meta{
					Id: m.Id{
						Root: file.Root,
						Name: name,
					},
					Size:    file.Size,
					ModTime: file.ModTime,
				},
				Kind:  m.FileFolder,
				State: m.Initial,
			}
			c.addFile(folderEntry)
		}

		name = name.Path.ParentName()
	}
}

func (c *controller) addCopyFile(file *m.File) {
	if newName, ok := c.resolveName(file); ok {
		c.rename(file, newName)
	}
	c.addFile(file)
}

func (c *controller) rename(file *m.File, newName m.Name) {
	nh := namehash{name: file.Name, hash: file.Hash}
	if _, ok := c.renames[nh]; !ok {
		c.archives[c.origin].scanner.Send(m.RenameFile{
			Hash: file.Hash,
			From: m.Id{Root: file.Root, Name: file.Name},
			To:   m.Id{Root: file.Root, Name: newName},
		})
	}
	file.Name = newName
	c.renames[nh] = file.Base
}

func (c *controller) resolveName(file *m.File) (m.Name, bool) {
	log.Printf("resolveName: name: %q", file)
	parts := strings.Split(file.Path.String(), "/")
	log.Printf("resolveName: parts: %s", parts)
	for i := 1; i < len(parts); i++ {
		path := m.Path(strings.Join(parts[:i], "/"))
		base := m.Base(parts[i])
		newId := m.Id{Root: file.Root, Name: m.Name{Path: path, Base: base}}
		log.Printf("resolveName: path: %q, base: %q, name: %q", path, base, newId)
		if _, ok := c.byId[newId]; ok {
			// TODO Don't ignore Hashes in c.byName
			newBase := c.uniqueBase(newId)
			log.Printf("resolveName: new base: %q", newBase)
			parts[i] = newBase.String()
			log.Printf("resolveName: parts: %s", parts)
			newId.Path = m.Path(strings.Join(parts, "/"))
			newId.Base = newId.Base

			return newId.Name, true
		}
	}
	if _, ok := c.byId[file.Id]; ok {
		newBase := c.uniqueBase(file.Name)
		return m.Name{Path: file.Path, Base: newBase}, true
	}
	return file.Name, false
}

func (c *controller) setInitialSelection(path m.Path) {
	c.currentPath = path
	c.selectFirst()
	for _, entry := range c.currentFolder().entries {
		if entry.Kind == m.FileFolder {
			c.setInitialSelection(m.Path(entry.Name.String()))
		}
	}
}

func (c *controller) removeEntry(id m.Id) {
	log.Panic("Implement controller.remove()")
	c.updateFolderStates("")
}

func (c *controller) handleHashingProgress(event m.HashingProgress) {
	c.archives[event.Root].fileHashed = event.Hashed
}

func (c *controller) handleCopyingProgress(event m.CopyingProgress) {
	c.fileCopiedSize = uint64(event)
}

func (c *controller) fileDeleted(event m.FileDeleted) {
	log.Printf("### %s", event)
	c.pendingFiles--
}

func (c *controller) fileRenamed(event m.FileRenamed) {
	log.Printf("### %s", event)
	c.setState(event.Hash, m.Resolved)
	c.pendingFiles--
}

func (c *controller) fileCopied(event m.FileCopied) {
	log.Printf("### %s", event)
	c.setState(event.Hash, m.Resolved)
	c.pendingFiles--
	c.fileCopiedSize = 0
	file := c.byId[event.From]
	c.totalCopiedSize += file.Size
	if c.totalCopiedSize == c.copySize {
		c.totalCopiedSize, c.copySize = 0, 0
	}
}

func (c *controller) setState(hash m.Hash, state m.State) {
	for _, entry := range c.byHash[hash] {
		entry.State = state
	}
	c.updateFolderStates("")
}

func (c *controller) updateFolderStates(path m.Path) m.State {
	// TODO: add updates for the Size and the ModTime as well
	state := m.Initial
	folder := c.getFolder(path)
	for _, entry := range folder.entries {
		if entry.Kind == m.FileFolder {
			state = mergeState(state, c.updateFolderStates(m.Path(entry.Name.String())))
			entry.State = state
		} else {
			state = mergeState(state, entry.State)
		}
	}
	return state
}

func mergeState(state1, state2 m.State) m.State {
	if state1 > state2 {
		return state1
	}
	return state2
}

func (c *controller) uniqueBase(file *m.File) m.Base {
	parts := strings.Split(file.Base.String(), ".")

	var part string
	if len(parts) == 1 {
		part = stripIdx(parts[0])
	} else {
		part = stripIdx(parts[len(parts)-2])
	}
	for idx := 1; ; idx++ {
		var newBase m.Base
		if len(parts) == 1 {
			newBase = m.Base(fmt.Sprintf("%s [%d]", part, idx))
		} else {
			parts[len(parts)-2] = fmt.Sprintf("%s [%d]", part, idx)
			newBase = m.Base(strings.Join(parts, "."))
		}
		newId := m.Id{Root: file.Root, Name: m.Name{Path: file.Path, Base: m.Base(newBase)}}
		if _, ok := c.byId[newId]; !ok {
			// c.allNames[newName] = struct{}{}
			// TODO add folder to allNames
			return newBase
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

func stripIdx(file string) string {
	state := expectCloseBracket
	i := len(file) - 1
	for ; i >= 0; i-- {
		ch := file[i]
		if ch == ']' && state == expectCloseBracket {
			state = expectDigit
		} else if ch >= '0' && ch <= '9' && (state == expectDigit || state == expectDigitOrOpenBracket) {
			state = expectDigitOrOpenBracket
		} else if ch == '[' && state == expectDigitOrOpenBracket {
			state = expectSpace
		} else if ch == ' ' && state == expectSpace {
			break
		} else {
			return file
		}
	}
	return file[:i]
}
