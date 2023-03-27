package files

import "time"

type FS interface {
	IsValid(path string) bool
	Scan(path string) <-chan any
	Stop()
}

type ScanState struct {
	Archive   string
	Name      string
	Remaining time.Duration
	Progress  float64
}

type ArchiveInfo struct {
	Archive string
	Files   []FileInfo
}

type FileInfo struct {
	Ino     uint64
	Archive string
	Name    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanError struct {
	Archive string
	Name    string
	Error   error
}
