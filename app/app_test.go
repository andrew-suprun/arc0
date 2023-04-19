package app

import (
	"arch/files"
	"testing"
	"time"
)

func TestApp(t *testing.T) {
	infos := []files.FileInfo{
		{
			Name:    "a/b/c/d.txt",
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
			Hash:    "ssss",
		},
	}
	app := app{}
	app.analizeArchive(infos)
}
