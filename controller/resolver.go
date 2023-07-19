package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"path/filepath"
	"strings"
)

type namehash struct {
	name string
	hash m.Hash
}

func (c *controller) autoresolve() {
	allNames := map[string]struct{}{}
	renamings := map[namehash]m.Name{}
	pending := map[m.Hash]struct{}{}
	originNames := map[string]m.Hash{}
	c.every(func(file *m.File) {
		if file.Root == c.origin {
			originNames[file.Name.String()] = file.Hash
		}
		allNames[file.Name.String()] = struct{}{}
		if file.Path == "" {
			return
		}
		parts := strings.Split(file.Path.String(), "/")
		for len(parts) > 0 {
			name := filepath.Join(parts...)
			if file.Root == c.origin {
				originNames[name] = ""
			}
			allNames[name] = struct{}{}
			parts = parts[:len(parts)-1]
		}
	})
	c.every(func(file *m.File) {
		if file.Root == c.origin {
			return
		}
		if originHash, ok := originNames[file.Name.String()]; ok && originHash != file.Hash {
			newName := uniqueName(allNames, renamings, file.Name, file.Hash)
			newId := m.Id{Root: file.Root, Name: newName}
			c.archives[c.origin].scanner.Send(m.RenameFile{From: file.Id, To: newId, Hash: file.Hash})
			file.Id = newId
			allNames[newId.Name.String()] = struct{}{}
			pending[file.Hash] = struct{}{}
		}
	})

	for _, files := range c.files {
		originFiles := []*m.File{}
		names := map[m.Name]struct{}{}
		for _, file := range files {
			if file.Root == c.origin {
				originFiles = append(originFiles, file)
			}
			names[file.Name] = struct{}{}
		}
		if len(originFiles) == 1 && (len(files) != len(c.roots) || len(names) != 1) {
			c.keepFile(originFiles[0])
		}
	}
	for hash := range pending {
		c.state[hash] = w.Pending
	}
}

func uniqueName(allNames map[string]struct{}, renamings map[namehash]m.Name, name m.Name, hash m.Hash) m.Name {
	nh := namehash{name.String(), hash}
	if newName, ok := renamings[nh]; ok {
		return newName
	}
	parts := strings.Split(name.Base.String(), ".")

	var part string
	if len(parts) == 1 {
		part = stripIdx(parts[0])
	} else {
		part = stripIdx(parts[len(parts)-2])
	}
	for idx := 1; ; idx++ {
		var newBase string
		if len(parts) == 1 {
			newBase = fmt.Sprintf("%s [%d]", part, idx)
		} else {
			parts[len(parts)-2] = fmt.Sprintf("%s [%d]", part, idx)
			newBase = strings.Join(parts, ".")
		}

		newName := m.Name{Path: name.Path, Base: m.Base(newBase)}
		if _, ok := allNames[newName.String()]; !ok {
			allNames[newName.String()] = struct{}{}
			renamings[nh] = newName
			return newName
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
