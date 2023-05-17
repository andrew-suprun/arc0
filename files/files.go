package files

import "time"

type FS interface {
	IsValid(path string) bool
	Scan(path string) <-chan Event
	Stop()
}

type Event interface {
	event()
}

type ScanState struct {
	Archive   string
	Name      string
	Remaining time.Duration
	Progress  float64
}

func (e *ScanState) event() {}

type ArchiveInfo struct {
	Archive string
	Files   FileInfos
}

func (e *ArchiveInfo) event() {}

type FileInfo struct {
	Ino     uint64
	Archive string
	Name    string
	Size    int
	ModTime time.Time
	Hash    string
}

type FileInfos []*FileInfo

type ScanError struct {
	Archive string
	Name    string
	Error   error
}

func (e ScanError) event() {}
