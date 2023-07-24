package controller

import (
	m "arch/model"
	"fmt"
	"strings"
)

type namehash struct {
	name string
	hash m.Hash
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
