package file_fs

import (
	"arch/lifecycle"
	m "arch/model"
	"arch/stream"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type fileFs struct {
	events *stream.Stream[m.Event]
	lc     *lifecycle.Lifecycle
}

func NewFs(events *stream.Stream[m.Event], lc *lifecycle.Lifecycle) m.FS {
	fs := &fileFs{
		events: events,
		lc:     lc,
	}

	return fs
}

func (fs *fileFs) NewArchiveScanner(root m.Root) m.ArchiveScanner {
	s := &scanner{
		root:     root,
		events:   fs.events,
		commands: stream.NewStream[m.FileCommand](root.String()),
		lc:       fs.lc,
		files:    map[uint64]*m.Meta{},
		stored:   map[uint64]*m.Meta{},
		sent:     map[m.Id]struct{}{},
	}
	go s.handleEvents()
	return s
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
