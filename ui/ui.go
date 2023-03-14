package ui

import "time"

type FileState int

const (
	Archived FileState = iota
	OnlyOriginal
	NoOriginal
	Conflict
)

type Folder struct {
	Name        string
	Size        int
	State       FileState
	SubArchives Archive
	Files       Files
}

type Archive []Folder

type File struct {
	Name    string
	Size    int
	State   FileState
	ModTime time.Time
}

type Files []File

type ScanState struct {
	Archive       string
	Folder        string
	File          string
	ETA           time.Time
	RemainingTime time.Duration
	Progress      float64
}

type ScanStates []ScanState
