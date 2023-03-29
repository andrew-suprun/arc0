package app

import (
	"arch/files"
	"arch/ui"
	"fmt"
	"log"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type app struct {
	paths         []string
	fs            files.FS
	renderer      ui.Renderer
	width, height int
	locations     []location
	quit          bool
	lineOffset    int
	scanStates    []*files.ScanState
	scanResults   []*files.ArchiveInfo
	archives      []folder
	archiveIdx    int
}

type location struct {
	path       []string
	file       string
	lineOffset int
}

type folder struct {
	name       string
	size       int
	subFolders map[string]folder
	files      map[string]file
}

type file struct {
	name    string
	size    int
	modTime time.Time
	hash    string
}

func Run(paths []string, fs files.FS, renderer ui.Renderer) {
	app := &app{
		paths:       paths,
		fs:          fs,
		renderer:    renderer,
		locations:   make([]location, len(paths)),
		scanStates:  make([]*files.ScanState, len(paths)),
		scanResults: make([]*files.ArchiveInfo, len(paths)),
	}

	tcellChan := make(chan any)

	go func() {
		for {
			tcellChan <- app.renderer.PollEvent()
		}
	}()

	fsInput := make(chan any)

	for _, archive := range paths {
		go func(archive string) {
			for ev := range fs.Scan(archive) {
				fsInput <- ev
			}
		}(archive)
	}

	for !app.quit {
		select {
		case event := <-fsInput:
			app.handleFsEvent(event)
		case event := <-tcellChan:
			app.handleUiEvent(event)
		}
		app.render()
	}
	app.fs.Stop()
	app.renderer.Exit()
}

func (app *app) analize() {
	app.archives = make([]folder, len(app.paths))
	for i := range app.scanResults {
		archive := &app.archives[i]
		archive.name = app.paths[i]
		archive.subFolders = map[string]folder{}
		archive.files = map[string]file{}
		for _, info := range app.scanResults[i].Files {
			path := strings.Split(info.Name, "/")
			name := path[len(path)-1]
			path = path[:len(path)-1]
			current := archive
			current.size += info.Size
			for _, dir := range path {
				sub, ok := current.subFolders[dir]
				if !ok {
					sub = folder{subFolders: map[string]folder{}, files: map[string]file{}}
					current.subFolders[dir] = sub
				}
				sub.size += info.Size
				current.subFolders[dir] = sub
				current = &sub
			}
			current.files[name] = file{size: info.Size, modTime: info.ModTime, hash: info.Hash}
		}
		// printArchive(archive, "", "")
	}
}

func printArchive(archive *folder, name, prefix string) {
	log.Printf("%sD: %s [%v]", prefix, name, archive.size)
	for name, sub := range archive.subFolders {
		printArchive(&sub, name, prefix+"    ")
	}
	for name, file := range archive.files {
		log.Printf("    %sF: %s [%v] %s", prefix, name, file.size, file.hash)
	}
}

func (app *app) handleFsEvent(event any) {
	switch event := event.(type) {
	case *files.ScanState:
		for i := range app.paths {
			if app.paths[i] == event.Archive {
				app.scanStates[i] = event
				break
			}
		}

	case *files.ArchiveInfo:
		log.Println("ArchiveInfo", event.Archive)
		for i := range app.paths {
			if app.paths[i] == event.Archive {
				app.scanStates[i] = nil
				app.scanResults[i] = event
				break
			}
		}
		doneScanning := true
		for i := range app.paths {
			if app.scanResults[i] == nil {
				doneScanning = false
				break
			}
		}
		if doneScanning {
			app.analize()
		}

	default:
		log.Printf("### unhandled event %#v", event)
	}
}

func (app *app) handleUiEvent(event any) {
	switch ev := event.(type) {
	case ui.ResizeEvent:
		log.Printf("ResizeEvent: [%d:%d]", ev.Width, ev.Height)
		app.width, app.height = ev.Width, ev.Height
		app.render()
		app.renderer.Sync()
	case ui.KeyEvent:
		log.Printf("EventKey: name=%s '%c'", ev.Name, ev.Rune)
		if ev.Name == "Ctrl+C" {
			app.quit = true
		}
		r := ev.Rune
		if r >= '1' && r <= '9' {
			idx := int(r - '1')
			if idx < len(app.paths) {
				app.archiveIdx = idx
			}
		}

	case ui.MouseEvent:
		log.Printf("EventMouse: [%d:%d]", ev.Col, ev.Line)
	default:
	}
}

func (app *app) render() {
	b := newBuilder(app.width, app.height)
	app.drawTitle(b)
	app.drawScanStats(b)
	app.drawArchive(b)
	app.drawStatusLine(b)
	app.renderer.Render(b.getScreen())
}

func (app *app) drawTitle(b *builder) {
	b.setDefaultStyle(ui.StyleAppTitle)
	b.setLayout(
		field{size: 11},
		field{flex: true, style: ui.StyleArchiveName},
	)
	b.drawTexts(" АРХИВАТОР ", app.archiveName())
}

func (app *app) drawStatusLine(b *builder) {
	b.setLayout(field{flex: true, style: ui.StyleArchiveName})
	b.setLine(app.height - 1)
	b.drawTexts(" Status line will be here...")
}

func (app *app) archiveName() string {
	if app.archives == nil {
		return ""
	}
	log.Println("### Archive", app.archives[app.archiveIdx].name)
	return app.archives[app.archiveIdx].name
}

func (app *app) drawScanStats(b *builder) {
	if app.scanStates == nil {
		return
	}

	b.setDefaultStyle(ui.StyleFile)

	for i, state := range app.scanStates {
		if app.scanStates[i] == nil {
			continue
		}

		b.setLayout(field{size: 28}, field{flex: true}, field{size: 1})
		b.drawTexts(" Архив", state.Archive)
		b.drawTexts(" Каталог", filepath.Dir(state.Name))
		b.drawTexts(" Документ", filepath.Base(state.Name))
		b.drawTexts(" Ожидаемое Время Завершения", time.Now().Add(state.Remaining).Format(time.TimeOnly))
		b.drawTexts(" Время До Завершения", state.Remaining.Truncate(time.Second).String())
		b.setLayout(field{size: 28}, field{flex: true, style: ui.StyleProgressBar}, field{size: 1})
		b.drawLine(text(" Общий Прогресс"), progressBar(state.Progress))
		b.newLine()
	}
}

func (app *app) drawArchive(b *builder) {
	if app.archives == nil {
		return
	}

	b.setLayout(field{size: 8}, field{size: 1}, field{flex: true}, field{size: 1}, field{size: 19}, field{size: 1}, field{size: 17, align: right}, field{size: 1})

	b.setDefaultStyle(ui.StyleArchiveHeader)
	b.drawTexts(" Статус", " ", "Документ", " ", "Время Изменения", " ", "Размер", " ")

	archive := app.archives[app.archiveIdx]
	location := app.locations[app.archiveIdx]
	for _, dir := range location.path {
		archive = archive.subFolders[dir]
	}

	// subfolders
	b.setDefaultStyle(ui.StyleFolder)
	subFolders := make([]folder, 0, len(archive.subFolders))
	for name, folder := range archive.subFolders {
		folder.name = name
		subFolders = append(subFolders, folder)
	}
	sort.Slice(subFolders, func(i, j int) bool {
		return subFolders[i].name < subFolders[j].name
	})
	for _, subFolder := range subFolders {
		b.drawTexts("", "", subFolder.name, " ", "Каталог", " ", formatSize(subFolder.size), " ")

		if app.lineOffset >= app.height-1 {
			break
		}
	}

	// files
	b.setDefaultStyle(ui.StyleFile)
	files := make([]file, 0, len(archive.files))
	for name, file := range archive.files {
		file.name = name
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})
	for _, file := range files {
		b.drawTexts("", "", file.name, " ", file.modTime.Format(time.DateTime), " ", formatSize(file.size), " ")

		if app.lineOffset >= app.height-1 {
			break
		}
	}
	// status line
}

func formatSize(size int) string {
	str := fmt.Sprintf("%13d", size)
	slice := []string{str[:1], str[1:4], str[4:7], str[7:10]}
	b := strings.Builder{}
	for _, s := range slice {
		b.WriteString(s)
		if s == " " || s == "   " {
			b.WriteString(" ")
		} else {
			b.WriteString(",")
		}
	}
	b.WriteString(str[10:])
	return b.String()
}
