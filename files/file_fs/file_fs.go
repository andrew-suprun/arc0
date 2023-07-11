package file_fs

import (
	"arch/actor"
	"arch/lifecycle"
	m "arch/model"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type fileFs struct {
	events m.EventChan
	lc     *lifecycle.Lifecycle
}

func NewFs(events m.EventChan, lc *lifecycle.Lifecycle) m.FS {
	fs := &fileFs{
		events: events,
		lc:     lc,
	}

	return fs
}

func (fs *fileFs) NewArchiveScanner(root m.Root) m.ArchiveScanner {
	s := &scanner{
		root:   root,
		events: fs.events,
		lc:     fs.lc,
		infos:  map[uint64]*fileInfo{},
	}
	s.Actor = actor.NewActor[m.FileCommand](s.handleCommand)
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
