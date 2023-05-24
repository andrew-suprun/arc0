package file_fs

import (
	"arch/lifecycle"
	"arch/model"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type file_fs struct {
	archivePath string
	events      model.EventHandler
	lc          *lifecycle.Lifecycle
}

func NewFs(path string, events model.EventHandler, lc *lifecycle.Lifecycle) (m model.FS, err error) {
	path, err = abs(path)
	if err != nil {
		return nil, err
	}
	return &file_fs{
		archivePath: path,
		events:      events,
		lc:          lc,
	}, nil
}

func abs(path string) (string, error) {
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

func (fs *file_fs) Scan() {
	go fs.scan()
}
