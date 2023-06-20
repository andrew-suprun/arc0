package controller

import (
	"arch/model"
	"sort"
	"strings"
)

func (m *controller) sort() {
	folder := m.folders[m.currentPath]
	files := sliceBy(folder.entries)
	var slice sort.Interface
	switch folder.sortColumn {
	case sortByName:
		slice = sliceByName{sliceBy: files}
	case sortByStatus:
		slice = sliceByStatus{sliceBy: files}
	case sortByTime:
		slice = sliceByTime{sliceBy: files}
	case sortBySize:
		slice = sliceBySize{sliceBy: files}
	}
	if !folder.sortAscending[folder.sortColumn] {
		slice = sort.Reverse(slice)
	}
	sort.Sort(slice)
}

type sliceBy model.Files

func (s sliceBy) Len() int {
	return len(s)
}

func (s sliceBy) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

type sliceByName struct {
	sliceBy
}

func (s sliceByName) Less(i, j int) bool {
	iName := strings.ToLower(s.sliceBy[i].Name)
	jName := strings.ToLower(s.sliceBy[j].Name)
	if iName < jName {
		return true
	} else if iName > jName {
		return false
	}
	if s.sliceBy[i].Size < s.sliceBy[j].Size {
		return true
	} else if s.sliceBy[i].Size > s.sliceBy[j].Size {
		return false
	}
	return s.sliceBy[i].ModTime.Before(s.sliceBy[j].ModTime)
}

type sliceByStatus struct {
	sliceBy
}

func (s sliceByStatus) Less(i, j int) bool {
	if s.sliceBy[i].Status < s.sliceBy[j].Status {
		return true
	} else if s.sliceBy[i].Status > s.sliceBy[j].Status {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) > strings.ToLower(s.sliceBy[j].Name)
}

type sliceByTime struct {
	sliceBy
}

func (s sliceByTime) Less(i, j int) bool {
	if s.sliceBy[i].ModTime.Before(s.sliceBy[j].ModTime) {
		return true
	} else if s.sliceBy[i].ModTime.After(s.sliceBy[j].ModTime) {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) < strings.ToLower(s.sliceBy[j].Name)
}

type sliceBySize struct {
	sliceBy
}

func (s sliceBySize) Less(i, j int) bool {
	if s.sliceBy[i].Size < s.sliceBy[j].Size {
		return true
	} else if s.sliceBy[i].Size > s.sliceBy[j].Size {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) < strings.ToLower(s.sliceBy[j].Name)
}
