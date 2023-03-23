package app

import (
	"log"
	"strings"
	"time"
)

type FS interface {
	IsValid(path string) bool
	Scan(path string) <-chan any
	Stop()
}

type UI interface {
	Run(app *App)
}

type App struct {
	Paths       []string
	Fs          FS
	ScanStates  []ScanState
	ScanResults []*ArchiveInfo
	ScanStarted time.Time
	Archives    []Folder
	ArchiveIdx  int
	UiInput     chan any
}

type Folder struct {
	Size       int
	SubFolders map[string]Folder
	Files      map[string]File
}

type File struct {
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanState struct {
	Archive     string
	Name        string
	Size        int
	Hashed      int
	TotalSize   int
	TotalToHash int
	TotalHashed int
}

type ArchiveInfo struct {
	Archive string
	Files   []FileInfo
}

type FileInfo struct {
	Ino     uint64
	Archive string
	Name    string
	Size    int
	ModTime time.Time
	Hash    string
}

type ScanError struct {
	Archive string
	Name    string
	Error   error
}

var nilTime time.Time

func NewApp(paths []string, fs FS) *App {
	app := &App{
		Paths:       paths,
		Fs:          fs,
		ScanStates:  make([]ScanState, len(paths)),
		ScanResults: make([]*ArchiveInfo, len(paths)),
		UiInput:     make(chan any, 1),
	}
	app.scan()
	return app
}

func (app *App) scan() {
	for _, path := range app.Paths {
		path := path
		scanChan := app.Fs.Scan(path)
		go func() {
			for scanEvent := range scanChan {
				if app.ScanStarted == nilTime {
					app.ScanStarted = time.Now()
				}
				select {
				case event := <-app.UiInput:
					switch event.(type) {
					case ScanState:
						// Drop previous []ScanState event, if any
					default:
						app.UiInput <- event
					}
				default:
				}

				app.UiInput <- scanEvent
			}
		}()
	}
}

func (app *App) Analize() {
	app.Archives = make([]Folder, len(app.Paths))
	for i := range app.ScanResults {
		archive := &app.Archives[i]
		archive.SubFolders = map[string]Folder{}
		archive.Files = map[string]File{}
		for _, info := range app.ScanResults[i].Files {
			log.Printf(" INFO: %s [%v]", info.Name, info.Size)
			path := strings.Split(info.Name, "/")
			name := path[len(path)-1]
			path = path[:len(path)-1]
			current := archive
			current.Size += info.Size
			for _, dir := range path {
				sub, ok := current.SubFolders[dir]
				if !ok {
					sub = Folder{SubFolders: map[string]Folder{}, Files: map[string]File{}}
					current.SubFolders[dir] = sub
				}
				sub.Size += info.Size
				current.SubFolders[dir] = sub
				current = &sub
			}
			current.Files[name] = File{Size: info.Size, ModTime: info.ModTime, Hash: info.Hash}
		}
		printArchive(archive, "", "")
	}
}

func printArchive(archive *Folder, name, prefix string) {
	log.Printf("%sD: %s [%v]", prefix, name, archive.Size)
	for name, sub := range archive.SubFolders {
		printArchive(&sub, name, prefix+"    ")
	}
	for name, file := range archive.Files {
		log.Printf("    %sF: %s [%v] %s", prefix, name, file.Size, file.Hash)
	}
}
