package app

import (
	"arch/files"
	"log"
	"testing"
	"time"
)

func TestApp(t *testing.T) {

	original := files.FileInfos{
		{
			Name:    "a/b/c/x.txt",
			Size:    100,
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
	}
	copy := files.FileInfos{
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
			Size:    400,
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
			Name:    "z.txt",
			Size:    200,
			ModTime: time.Now(),
			Hash:    "hhhh",
		},
	}
	archives := files.ArchiveInfos{
		{
			Archive: "Original",
			Files:   original,
		},
		{
			Archive: "Copy",
			Files:   copy,
		},
	}
	app := &app{
		scanResults: archives,
	}
	app.analizeArchives()

	log.Printf("\n%v", app.analisys[0])
}
