package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"strings"
)

type archive struct {
	scanner       m.ArchiveScanner
	files         map[m.FullName]*w.File
	pending       map[m.FullName]*w.File
	progress      m.Progress
	progressState m.ProgressState
	totalSize     uint64
	totalHashed   uint64
	copySize      uint64
	totalCopied   uint64
}

func (a *archive) update(proc func(file *w.File)) {
	for _, entry := range a.files {
		proc(entry)
	}
}

func (a *archive) fileByFullName(name m.FullName) *w.File {
	return a.files[name]
}

func (a *archive) fileByNewName(name m.FullName) *w.File {
	if result, ok := a.pending[name]; ok {
		return result
	}
	result := a.files[name]
	if result != nil && !result.Pending {
		return result
	}
	return nil
}

func (a *archive) ensureNameAvailable(id m.FileId) *m.RenameFile {
	file := a.fileByNewName(id.FullName())
	if file != nil {
		newName := a.newName(id.FullName())
		file.PendingName = newName
		a.pending[newName] = file
		return &m.RenameFile{FileId: id, NewFullName: newName}
	}
	return nil
}

func (a *archive) newName(name m.FullName) m.FullName {
	parts := strings.Split(name.Name.String(), ".")

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
			if name.Path == entity.Path && newName == entity.Name.String() {
				exists = true
				break
			}
		}
		if !exists {
			return m.FullName{Path: name.Path, Name: m.Name(newName)}
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
