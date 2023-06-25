package controller

import (
	"arch/model"
	"sort"
	"strings"
)

func (f *folder) sort() {
	files := sliceBy(f.entries)
	var slice sort.Interface
	switch f.sortColumn {
	case sortByName:
		slice = sliceByName{sliceBy: files}
	case sortByStatus:
		slice = sliceByStatus{sliceBy: files}
	case sortByTime:
		slice = sliceByTime{sliceBy: files}
	case sortBySize:
		slice = sliceBySize{sliceBy: files}
	}
	if !f.sortAscending[f.sortColumn] {
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
	iStatus := s.sliceBy[i].Status
	jStatus := s.sliceBy[j].Status
	if iStatus < jStatus {
		return true
	} else if iStatus > jStatus {
		return false
	}

	return s.sliceBy[i].ModTime.Before(s.sliceBy[j].ModTime)
}

type sliceByStatus struct {
	sliceBy
}

func (s sliceByStatus) Less(i, j int) bool {
	iStatus := s.sliceBy[i].Status
	jStatus := s.sliceBy[j].Status
	if iStatus < jStatus {
		return true
	} else if iStatus > jStatus {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) > strings.ToLower(s.sliceBy[j].Name)
}

type sliceByTime struct {
	sliceBy
}

func (s sliceByTime) Less(i, j int) bool {
	iModTime := s.sliceBy[i].ModTime
	jModTime := s.sliceBy[j].ModTime
	if iModTime.Before(jModTime) {
		return true
	} else if iModTime.After(jModTime) {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) < strings.ToLower(s.sliceBy[j].Name)
}

type sliceBySize struct {
	sliceBy
}

func (s sliceBySize) Less(i, j int) bool {
	iSize := s.sliceBy[i].Size
	jSize := s.sliceBy[j].Size
	if iSize < jSize {
		return true
	} else if iSize > jSize {
		return false
	}

	return strings.ToLower(s.sliceBy[i].Name) < strings.ToLower(s.sliceBy[j].Name)
}