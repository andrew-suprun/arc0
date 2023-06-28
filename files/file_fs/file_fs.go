package file_fs

import (
	"arch/actor"
	"arch/lifecycle"
	"arch/model"
	"os"
	"path/filepath"

	"golang.org/x/text/unicode/norm"
)

type fileFs struct {
	events  model.EventChan
	lc      *lifecycle.Lifecycle
	handler actor.Actor[model.HandleFiles]
}

func NewFs(events model.EventChan, lc *lifecycle.Lifecycle) model.FS {
	fs := &fileFs{
		events: events,
		lc:     lc,
	}
	fs.handler = actor.NewActor[model.HandleFiles](fs.handleFiles)

	return fs
}

func (fs *fileFs) NewArchiveScanner(root string) model.ArchiveScanner {
	return &scanner{
		root:   root,
		events: fs.events,
		lc:     fs.lc,
		infos:  map[uint64]*fileInfo{},
	}
}

func (fs *fileFs) Send(cmd model.HandleFiles) {
	fs.handler.Send(cmd)
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
