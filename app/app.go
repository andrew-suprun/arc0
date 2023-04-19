package app

import (
	"arch/files"
	"arch/ui"
	"fmt"
	"log"
	"path/filepath"
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
	scanStates    []*files.ScanState
	scanResults   []*files.ArchiveInfo
	archives      []*file
	archiveIdx    int
}

type location struct {
	path       []string
	file       string
	lineOffset int
}

type fileKind int

const (
	regular fileKind = iota
	folder
)

type file struct {
	kind       fileKind
	name       string
	size       int
	modTime    time.Time
	hash       string
	subFolders []*file
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

func (app *app) analizeArchives() {
	app.archives = make([]*file, len(app.paths))
	for i := range app.scanResults {
		app.archives[i] = app.analizeArchive(app.scanResults[i].Files)
		app.archives[i].name = app.paths[i] // ???
	}
}

func (app *app) analizeArchive(infos []files.FileInfo) *file {
	archive := &file{kind: folder}
	for _, info := range infos {
		path := strings.Split(info.Name, "/")
		name := path[len(path)-1]
		path = path[:len(path)-1]
		current := archive
		current.size += info.Size
		for _, dir := range path {
			sub := subFolder(current, dir)
			sub.size += info.Size
			current = sub
		}
		current.subFolders = append(current.subFolders, &file{
			kind:    regular,
			name:    name,
			size:    info.Size,
			modTime: info.ModTime,
			hash:    info.Hash,
		})
	}
	printArchive(archive, "", "")
	return archive
}

func subFolder(dir *file, name string) *file {
	for i := range dir.subFolders {
		if name == dir.subFolders[i].name {
			return dir.subFolders[i]
		}
	}
	subFolder := &file{kind: folder, name: name}
	dir.subFolders = append(dir.subFolders, subFolder)
	return subFolder
}

func printArchive(archive *file, name, prefix string) {
	kind := "D"
	if archive.kind == regular {
		kind = "F"
	}
	log.Printf("%s%s: %s [%v]", prefix, kind, name, archive.size)
	for _, file := range archive.subFolders {
		printArchive(file, file.name, prefix+"│ ")
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
			app.analizeArchives()
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
	app.drawHeaderView()
	app.drawScanStats()
	app.drawArchive()

	app.drawStatusLine()
	app.renderer.Show()
}

func (app *app) drawHeaderView() {
	view := ui.View(0, 0, app.width, 1,
		ui.Layout([]ui.Field{ui.Fixed(11), ui.Flex(1)},
			ui.Line(
				ui.Styled(ui.StyleAppTitle, ui.Text(" АРХИВАТОР ")),
				ui.Styled(ui.StyleArchiveName, ui.Text(app.archiveName())),
			),
		),
	)
	app.renderer.Render(view...)
}

func (app *app) archiveName() string {
	if app.archives == nil {
		return ""
	}
	log.Println("### Archive", app.archives[app.archiveIdx].name)
	return app.archives[app.archiveIdx].name
}

func (app *app) drawScanStats() {
	if app.scanStates == nil {
		return
	}

	contents := []ui.Widget{}
	for i, state := range app.scanStates {
		if app.scanStates[i] == nil {
			continue
		}

		contents = append(contents,
			ui.Line(ui.Text("Архив"), ui.Text(state.Archive)),
			ui.Line(ui.Text("Каталог"), ui.Text(filepath.Dir(state.Name))),
			ui.Line(ui.Text("Документ"), ui.Text(filepath.Base(state.Name))),
			ui.Line(ui.Text("Ожидаемое Время Завершения"), ui.Text(time.Now().Add(state.Remaining).Format(time.TimeOnly))),
			ui.Line(ui.Text("Время До Завершения"), ui.Text(time.Now().Add(state.Remaining.Truncate(time.Second)).String())),
			ui.Line(ui.Text("Общий Прогресс"), ui.Styled(ui.StyleProgressBar, ui.ProgressBar(state.Progress))),
			ui.Line(ui.Text(""), ui.Text("")),
		)
	}

	for len(contents) < app.height-2 {
		contents = append(contents, ui.Line(ui.Text(""), ui.Text(""), ui.Text("")))
	}

	view := ui.View(0, 1, app.width, app.height-2,
		ui.Styled(
			ui.StyleDefault,
			ui.Layout([]ui.Field{ui.Pad(" "), ui.Fixed(26), ui.Pad(" "), ui.Flex(1), ui.Pad(" ")}, contents...)),
	)
	app.renderer.Render(view...)
}

func (app *app) drawArchive() {
	if app.archives == nil {
		return
	}

	contents := []ui.Widget{
		ui.Styled(ui.StyleArchiveHeader,
			ui.Line(ui.Text("Статус"), ui.Text("Документ"), ui.Text("Время Изменения"), ui.Text("Размер")),
		),
	}

	// archive := app.archives[app.archiveIdx]
	// location := app.locations[app.archiveIdx]
	// for _, dir := range location.path {
	// 	archive = archive.subFolders[dir]
	// }

	// // subfolders
	// b.SetStyle(ui.StyleFolder)
	// subFolders := make([]folder, 0, len(archive.subFolders))
	// for name, folder := range archive.subFolders {
	// 	folder.name = name
	// 	subFolders = append(subFolders, folder)
	// }
	// sort.Slice(subFolders, func(i, j int) bool {
	// 	return subFolders[i].name < subFolders[j].name
	// })
	// for _, subFolder := range subFolders {
	// 	b.AddText("")
	// 	b.AddText(" " + subFolder.name)
	// 	b.AddText(" Каталог")
	// 	b.AddText(formatSize(subFolder.size))
	// 	b.LayoutLine()

	// 	if app.lineOffset >= app.height-1 {
	// 		break
	// 	}
	// }

	// // files
	// b.SetStyle(ui.StyleFile)
	// files := make([]file, 0, len(archive.files))
	// for name, file := range archive.files {
	// 	file.name = name
	// 	files = append(files, file)
	// }
	// sort.Slice(files, func(i, j int) bool {
	// 	return files[i].name < files[j].name
	// })
	// for _, file := range files {
	// 	b.AddText("")
	// 	b.AddText(" " + file.name)
	// 	b.AddText(" " + file.modTime.Format(time.DateTime))
	// 	b.AddText(formatSize(file.size))
	// 	b.LayoutLine()

	// 	if app.lineOffset >= app.height-1 {
	// 		break
	// 	}
	// }

	view := ui.View(0, 1, app.width, app.height-2,
		ui.Styled(ui.StyleDefault,
			ui.Layout([]ui.Field{ui.Pad(" "), ui.Fixed(7), ui.Pad(" "), ui.Flex(1), ui.Pad(" "), ui.Fixed(19), ui.Pad(" "), ui.Fixed(16), ui.Pad(" ")},
				contents...,
			),
		),
	)
	app.renderer.Render(view...)
}

func (app *app) drawStatusLine() {
	view := ui.View(0, app.height-1, app.width, 1,
		ui.Styled(ui.StyleArchiveName,
			ui.Layout([]ui.Field{ui.Pad(" "), ui.Flex(1), ui.Pad(" ")},
				ui.Line(ui.Text("Status line will be here...")),
			),
		),
	)
	log.Printf("status line: %#v", view)
	app.renderer.Render(view...)
}

func formatSize(size int) string {
	str := fmt.Sprintf("%13d ", size)
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
