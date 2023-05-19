package file_fs

import (
	"arch/files"
	"arch/lifecycle"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type file_fs struct {
	lc *lifecycle.Lifecycle
}

func NewFs() files.FS {
	return &file_fs{
		lc: lifecycle.New(),
	}
}

func (fs *file_fs) Abs(path string) (string, error) {
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

func (fs *file_fs) Scan(path string) <-chan files.Event {
	result := make(chan files.Event, 1)
	go fs.scan(path, result)
	return result
}

func (fs *file_fs) Stop() {
	fs.lc.Stop()
}
