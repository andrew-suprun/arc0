package msg

import (
	"time"
)

type CmdScan struct {
	Base string
}

type CmdQuit struct{}

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

type ScanMetas struct {
	Base  string
	Metas []*FileMeta
}

type Analysis [][]ScanMetas

type ScanDone struct {
	Base string
}

type QuitApp struct{}
