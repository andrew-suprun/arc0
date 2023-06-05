package file_fs

import (
	"arch/events"
	"arch/files"
	"arch/lifecycle"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type file_fs struct {
	events events.EventChan
	lc     *lifecycle.Lifecycle
}

func NewFs(events events.EventChan, lc *lifecycle.Lifecycle) files.FS {
	return &file_fs{
		events: events,
		lc:     lc,
	}
}

func (fs *file_fs) NewScanner(archivePath string) files.Scanner {
	return &scanner{
		events:      fs.events,
		lc:          fs.lc,
		archivePath: archivePath,
		infos:       map[uint64]*fileInfo{},
	}
}

func AbsPath(path string) (string, error) {
	var err error
	path, err = filepath.Abs(path)
	path = norm.NFC.String(path)
	if err != nil {
		return "", err
	}

	_, err = os.Stat(path)
	if err != nil {
		return "", err
	}
	return path, nil
}
