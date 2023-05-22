package files

import "time"

type FS interface {
	Abs(path string) (string, error)
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
	Files   []*FileInfo
}

func (e *ArchiveInfo) event() {}

type FileInfo struct {
	Archive string
	Path    string
	Name    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanError struct {
	Archive string
	Path    string
	Error   error
}

func (e ScanError) event() {}
