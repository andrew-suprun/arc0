package mock2_fs

import (
	"arch/model"
	"time"
)

type mockFs struct {
	events model.EventHandler
}

func NewFs(path string, events model.EventHandler) model.FS {
	return &mockFs{
		events: events,
	}
}

func (fsys *mockFs) Scan(archivePath string) error {
	fsys.events <- func(m *model.Model) {
		switch archivePath {
		case "origin":
			m.Archives[0].Files = origin
		case "copy 1":
			m.Archives[0].Files = copy1
		case "copy 2":
			m.Archives[0].Files = copy2
		}
	}
	return nil
}

var origin = []*model.FileMeta{
	{
		Archive:  "origin",
		FullName: "a/b/c/x.txt",
		Size:     150,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "origin",
		FullName: "a/b/e/f.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "gggg",
	},
	{
		Archive:  "origin",
		FullName: "a/b/e/g.txt",
		Size:     300,
		ModTime:  time.Now(),
		Hash:     "tttt",
	},
	{
		Archive:  "origin",
		FullName: "x.txt",
		Size:     400,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "origin",
		FullName: "q/w/e/r/t/y.txt",
		Size:     400,
		ModTime:  time.Now(),
		Hash:     "qwerty",
	},
	{
		Archive:  "origin",
		FullName: "yyy.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "yyyy",
	},
}
var copy1 = []*model.FileMeta{
	{
		Archive:  "copy 1",
		FullName: "a/b/c/d.txt",
		Size:     100,
		ModTime:  time.Now(),
		Hash:     "llll",
	},
	{
		Archive:  "copy 1",
		FullName: "a/b/e/f.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "copy 1",
		FullName: "a/b/e/g.txt",
		Size:     300,
		ModTime:  time.Now(),
		Hash:     "tttt",
	},
	{
		Archive:  "copy 1",
		FullName: "x.txt",
		Size:     500,
		ModTime:  time.Now(),
		Hash:     "mmmm",
	},
	{
		Archive:  "copy 1",
		FullName: "y.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "gggg",
	},
	{
		Archive:  "copy 1",
		FullName: "a/b/c/x.txt",
		Size:     150,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "copy 1",
		FullName: "zzzz.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "copy 1",
		FullName: "x/y/z.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "zzzz",
	},
	{
		Archive:  "copy 1",
		FullName: "yyy.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "yyyy",
	},
}
var copy2 = []*model.FileMeta{
	{
		Archive:  "copy 2",
		FullName: "a/b/c/f.txt",
		Size:     150,
		ModTime:  time.Now(),
		Hash:     "hhhh",
	},
	{
		Archive:  "copy 2",
		FullName: "a/b/e/x.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "gggg",
	},
	{
		Archive:  "copy 2",
		FullName: "a/b/e/g.txt",
		Size:     300,
		ModTime:  time.Now(),
		Hash:     "tttt",
	},
	{
		Archive:  "copy 2",
		FullName: "x",
		Size:     4_000_000_000_000,
		ModTime:  time.Now(),
		Hash:     "asdfg",
	},
	{
		Archive:  "copy 2",
		FullName: "q/w/e/r/t/y.txt",
		Size:     300,
		ModTime:  time.Now(),
		Hash:     "12345",
	},
	{
		Archive:  "copy 2",
		FullName: "yyy.txt",
		Size:     200,
		ModTime:  time.Now(),
		Hash:     "yyyy",
	},
}
