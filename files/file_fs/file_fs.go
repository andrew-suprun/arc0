package file_fs

import (
	"arch/lifecycle"
	"arch/model"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type fileFs struct {
	events model.EventChan
	lc     *lifecycle.Lifecycle
}

func NewFs(events model.EventChan, lc *lifecycle.Lifecycle) model.FS {
	return &fileFs{
		events: events,
		lc:     lc,
	}
}

func (fs *fileFs) NewScanner(root string) model.Scanner {
	return &scanner{
		events: fs.events,
		lc:     fs.lc,
		root:   root,
		infos:  map[uint64]*fileInfo{},
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
