package api

import (
	"time"
)

type FileMeta struct {
	Ino     uint64
	Base    string
	Path    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanError struct {
	Base  string
	Path  string
	Error error
}

type ScanStat struct {
	Base        string
	Path        string
	Size        int
	Hashed      int
	TotalSize   int
	TotalToHash int
	TotalHashed int
}

type ScanDone struct {
	Base string
}

type QuitApp struct{}
