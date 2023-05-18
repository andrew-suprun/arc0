package ui

import (
	"log"
	"sort"
	"strings"
)

type sortColumn int

const (
	sortByName sortColumn = iota
	sortByStatus
	sortByTime
	sortBySize
)

type sortDirection int

const (
	asc sortDirection = iota
	desc
)

func (m *model) sort() {
	files := m.currentLocation().file.files
	log.Println("sortBy", m.sortColumn, "sortAscending", m.sortAscending[m.sortColumn])
	var slice sort.Interface
	switch m.sortColumn {
	case sortByName:
		slice = sliceByName{files}
	case sortByStatus:
		slice = sliceByStatus{files}
	case sortByTime:
		slice = sliceByTime{files}
	case sortBySize:
		slice = sliceBySize{files}
	}
	if !m.sortAscending[m.sortColumn] {
		slice = sort.Reverse(slice)
	}
	sort.Sort(slice)
}

type sliceBy []*fileInfo

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
	iName := strings.ToLower(s.sliceBy[i].name)
	jName := strings.ToLower(s.sliceBy[j].name)
	if iName < jName {
		return true
	} else if iName > jName {
		return false
	}
	return s.sliceBy[i].status < s.sliceBy[j].status
}

type sliceByStatus struct {
	sliceBy
}

func (s sliceByStatus) Less(i, j int) bool {
	if s.sliceBy[i].status < s.sliceBy[j].status {
		return true
	} else if s.sliceBy[i].status > s.sliceBy[j].status {
		return false
	}

	return strings.ToLower(s.sliceBy[i].name) < strings.ToLower(s.sliceBy[j].name)
}

type sliceByTime struct {
	sliceBy
}

func (s sliceByTime) Less(i, j int) bool {
	if s.sliceBy[i].modTime.Before(s.sliceBy[j].modTime) {
		return true
	} else if s.sliceBy[i].status > s.sliceBy[j].status {
		return false
	}

	return strings.ToLower(s.sliceBy[i].name) < strings.ToLower(s.sliceBy[j].name)
}

type sliceBySize struct {
	sliceBy
}

func (s sliceBySize) Less(i, j int) bool {
	if s.sliceBy[i].size < s.sliceBy[j].size {
		return true
	} else if s.sliceBy[i].size > s.sliceBy[j].size {
		return false
	}

	return strings.ToLower(s.sliceBy[i].name) < strings.ToLower(s.sliceBy[j].name)
}
