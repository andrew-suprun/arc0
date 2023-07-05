package controller

import (
	w "arch/widgets"
	"sort"
	"strings"
)

func (c *controller) sort() {
	if len(c.entries) == 0 {
		return
	}

	folder := c.folders[c.currentPath]
	files := sliceBy(c.entries)
	var slice sort.Interface
	switch folder.sortColumn {
	case w.SortByName:
		slice = sliceByName{sliceBy: files}
	case w.SortByStatus:
		slice = sliceByStatus{sliceBy: files}
	case w.SortByTime:
		slice = sliceByTime{sliceBy: files}
	case w.SortBySize:
		slice = sliceBySize{sliceBy: files}
	}
	if !folder.sortAscending[folder.sortColumn] {
		slice = sort.Reverse(slice)
	}
	sort.Sort(slice)
}

type sliceBy []*w.File

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
	iName := strings.ToLower(s.sliceBy[i].Name.String())
	jName := strings.ToLower(s.sliceBy[j].Name.String())
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

	iName := strings.ToLower(s.sliceBy[i].Name.String())
	jName := strings.ToLower(s.sliceBy[j].Name.String())
	if iName < jName {
		return true
	} else if iName > jName {
		return false
	}

	return s.sliceBy[i].Size < s.sliceBy[j].Size
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

	return strings.ToLower(s.sliceBy[i].Name.String()) < strings.ToLower(s.sliceBy[j].Name.String())
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

	return strings.ToLower(s.sliceBy[i].Name.String()) < strings.ToLower(s.sliceBy[j].Name.String())
}
