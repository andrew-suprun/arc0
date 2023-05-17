package mock2_fs

import (
	"arch/files"
	"time"
)

type mockFs struct{}

func NewFs() files.FS {
	return &mockFs{}
}

func (fs *mockFs) IsValid(path string) bool {
	return true
}

func (fs *mockFs) Stop() {
}

func (fsys *mockFs) Scan(path string) <-chan files.Event {
	if path == "origin" {
		result := make(chan files.Event, 1)
		result <- &files.ArchiveInfo{
			Archive: "origin",
			Files:   origin,
		}
		close(result)
		return result
	}
	if path == "copy 1" {
		result := make(chan files.Event, 1)
		result <- &files.ArchiveInfo{
			Archive: "copy 1",
			Files:   copy1,
		}
		close(result)
		return result
	}
	if path == "copy 2" {
		result := make(chan files.Event, 1)
		result <- &files.ArchiveInfo{
			Archive: "copy 2",
			Files:   copy2,
		}
		close(result)
		return result
	}
	return nil
}

var origin = files.FileInfos{
	{
		Name:    "a/b/c/x.txt",
		Size:    150,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "a/b/e/f.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "gggg",
	},
	{
		Name:    "a/b/e/g.txt",
		Size:    300,
		ModTime: time.Now(),
		Hash:    "tttt",
	},
	{
		Name:    "x.txt",
		Size:    400,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "q/w/e/r/t/y.txt",
		Size:    400,
		ModTime: time.Now(),
		Hash:    "qwerty",
	},
	{
		Name:    "yyy.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "yyyy",
	},
}
var copy1 = files.FileInfos{
	{
		Name:    "a/b/c/d.txt",
		Size:    100,
		ModTime: time.Now(),
		Hash:    "llll",
	},
	{
		Name:    "a/b/e/f.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "a/b/e/g.txt",
		Size:    300,
		ModTime: time.Now(),
		Hash:    "tttt",
	},
	{
		Name:    "x.txt",
		Size:    500,
		ModTime: time.Now(),
		Hash:    "mmmm",
	},
	{
		Name:    "y.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "gggg",
	},
	{
		Name:    "a/b/c/x.txt",
		Size:    150,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "zzzz.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "x/y/z.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "zzzz",
	},
	{
		Name:    "yyy.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "yyyy",
	},
}
var copy2 = files.FileInfos{
	{
		Name:    "a/b/c/f.txt",
		Size:    150,
		ModTime: time.Now(),
		Hash:    "hhhh",
	},
	{
		Name:    "a/b/e/x.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "gggg",
	},
	{
		Name:    "a/b/e/g.txt",
		Size:    300,
		ModTime: time.Now(),
		Hash:    "tttt",
	},
	{
		Name:    "x",
		Size:    4_000_000_000_000,
		ModTime: time.Now(),
		Hash:    "asdfg",
	},
	{
		Name:    "q/w/e/r/t/y.txt",
		Size:    300,
		ModTime: time.Now(),
		Hash:    "12345",
	},
	{
		Name:    "yyy.txt",
		Size:    200,
		ModTime: time.Now(),
		Hash:    "yyyy",
	},
}
