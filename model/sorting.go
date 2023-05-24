package model

import (
	"log"
	"sort"
	"strings"
)

func (m *Model) Sort() {
	files := sliceBy(m.CurerntFolder().File.Files)
	log.Println("sortBy", m.SortColumn, "sortAscending", m.SortAscending[m.SortColumn])
	var slice sort.Interface
	switch m.SortColumn {
	case SortByName:
		slice = sliceByName{sliceBy: files}
	case SortByStatus:
		slice = sliceByStatus{sliceBy: files}
	case SortByTime:
		slice = sliceByTime{sliceBy: files}
	case SortBySize:
		slice = sliceBySize{sliceBy: files}
	}
	if !m.SortAscending[m.SortColumn] {
		slice = sort.Reverse(slice)
	}
	sort.Sort(slice)
}

type sliceBy FileInfos

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
	iName := strings.ToLower(s.sliceBy[i].FullName)
	jName := strings.ToLower(s.sliceBy[j].FullName)
	if iName < jName {
		return true
	} else if iName > jName {
		return false
	}
	return s.sliceBy[i].Status < s.sliceBy[j].Status
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

	return strings.ToLower(s.sliceBy[i].FullName) > strings.ToLower(s.sliceBy[j].FullName)
}

type sliceByTime struct {
	sliceBy
}

func (s sliceByTime) Less(i, j int) bool {
	if s.sliceBy[i].ModTime.Before(s.sliceBy[j].ModTime) {
		return true
	} else if s.sliceBy[i].Status > s.sliceBy[j].Status {
		return false
	}

	return strings.ToLower(s.sliceBy[i].FullName) < strings.ToLower(s.sliceBy[j].FullName)
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

	return strings.ToLower(s.sliceBy[i].FullName) < strings.ToLower(s.sliceBy[j].FullName)
}
