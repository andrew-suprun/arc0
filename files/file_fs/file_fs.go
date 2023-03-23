package file_fs

import (
	"arch/app"
	"arch/lifecycle"
	"os"
)

type file_fs struct {
	lc *lifecycle.Lifecycle
}

func NewFs() app.FS {
	return &file_fs{
		lc: lifecycle.New(),
	}
}

func (fs *file_fs) IsValid(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (fsys *file_fs) Scan(path string) <-chan any {
	result := make(chan any, 1)
	go fsys.scan(path, result)
	return result
}

func (fsys *file_fs) Stop() {
	fsys.lc.Stop()
}
