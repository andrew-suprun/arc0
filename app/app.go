package app

import (
	"arch/files"
	"arch/ui"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muesli/ansi"
)

type app struct {
	paths       []string
	fs          files.FS
	renderer    ui.Renderer
	screen      ui.Screen
	locations   []location
	quit        bool
	lineOffset  int
	scanStates  []*files.ScanState
	scanResults []*files.ArchiveInfo
	archives    []folder
	ArchiveIdx  int
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
		app.screen = make([][]ui.Char, ev.Height)
		for line := range app.screen {
			app.screen[line] = make([]ui.Char, ev.Width)
		}
		log.Printf("EventResize: cols=%d lines=%d", ev.Width, ev.Height)
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
				app.ArchiveIdx = idx
			}
		}

	case ui.MouseEvent:
		log.Printf("EventMouse: [%d:%d]", ev.Col, ev.Line)
	default:
	}
}

func (app *app) render() {
	app.clear()
	app.drawTitle()
	app.drawScanStats()
	app.drawArchive()
	app.renderer.Render(app.screen)
}

func (app *app) clear() {
	for line := range app.screen {
		for col := range app.screen[line] {
			app.screen[line][col] = ui.Char{Rune: ' ', Style: ui.StyleFile}
		}
	}
	app.lineOffset = 0
}

func (app *app) drawTitle() {
	app.layoutLine(
		text{text: " АРХИВАТОР ", style: ui.StyleAppTitle},
		text{text: app.archiveName(), style: ui.StyleArchiveName, flex: true},
	)
}

func (app *app) archiveName() string {
	if app.archives == nil {
		return ""
	}
	log.Println("### Archive", app.archives[app.ArchiveIdx].name)
	return app.archives[app.ArchiveIdx].name
}

func (app *app) drawScanStats() {
	if app.scanStates == nil {
		return
	}
	for i, state := range app.scanStates {
		if app.scanStates[i] == nil {
			continue
		}

		app.drawFormLine(" Архив                      ", state.Archive)
		app.drawFormLine(" Каталог                    ", filepath.Dir(state.Name))
		app.drawFormLine(" Документ                   ", filepath.Base(state.Name))
		app.drawFormLine(" Ожидаемое Время Завершения ", time.Now().Add(state.Remaining).Format(time.TimeOnly))
		app.drawFormLine(" Время До Завершения        ", state.Remaining.Truncate(time.Second).String())
		app.layoutLine(
			text{text: " Общий Прогресс             ", style: ui.StyleFile},
			progressBar{value: state.Progress, style: ui.StyleProgressBar},
			text{text: " ", style: ui.StyleFile},
		)
		app.lineOffset++
	}
}

func (app *app) drawFormLine(name, value string) {
	app.layoutLine(
		text{text: name, style: ui.StyleFile},
		text{text: value, style: ui.StyleFolder, flex: true},
		text{text: " ", style: ui.StyleFile},
	)
}

func (app *app) drawArchive() {
	if app.archives == nil {
		return
	}
	app.layoutLine(
		text{text: " Статус", size: 8, style: ui.StyleArchiveHeader},
		text{text: " ", style: ui.StyleArchiveHeader},
		text{text: "Документ", flex: true, style: ui.StyleArchiveHeader},
		text{text: " ", style: ui.StyleArchiveHeader},
		text{text: "Время Изменения", size: 19, style: ui.StyleArchiveHeader},
		text{text: " ", style: ui.StyleArchiveHeader},
		text{text: "Размер", size: 17, align: right, style: ui.StyleArchiveHeader},
		text{text: " ", style: ui.StyleArchiveHeader},
	)
	archive := app.archives[app.ArchiveIdx]
	location := app.locations[app.ArchiveIdx]
	for _, dir := range location.path {
		archive = archive.subFolders[dir]
	}

	// subfolders
	subFolders := make([]folder, 0, len(archive.subFolders))
	for name, folder := range archive.subFolders {
		folder.name = name
		subFolders = append(subFolders, folder)
	}
	sort.Slice(subFolders, func(i, j int) bool {
		return subFolders[i].name < subFolders[j].name
	})
	for _, subFolder := range subFolders {
		app.layoutLine(
			text{text: "", size: 8, style: ui.StyleFolder},
			text{text: subFolder.name, flex: true, style: ui.StyleFolder},
			text{text: " ", style: ui.StyleFolder},
			text{text: "Каталог", size: 19, style: ui.StyleFolder},
			text{text: " ", style: ui.StyleFolder},
			text{text: formatSize(subFolder.size), size: 17, align: right, style: ui.StyleFolder},
			text{text: " ", style: ui.StyleFolder},
		)

		if app.lineOffset >= len(app.screen)-2 {
			break
		}
	}

	// files
	files := make([]file, 0, len(archive.files))
	for name, file := range archive.files {
		file.name = name
		files = append(files, file)
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].name < files[j].name
	})
	for _, file := range files {
		app.layoutLine(
			text{text: "", size: 8, style: ui.StyleFile},
			text{text: file.name, flex: true, style: ui.StyleFile},
			text{text: " ", style: ui.StyleFile},
			text{text: file.modTime.Format(time.DateTime), size: 19, style: ui.StyleFile},
			text{text: " ", style: ui.StyleFile},
			text{text: formatSize(file.size), size: 17, align: right, style: ui.StyleFile},
			text{text: " ", style: ui.StyleFile},
		)

		if app.lineOffset >= len(app.screen)-2 {
			break
		}
	}
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

type alignment byte

const (
	left alignment = iota
	right
)

func (app *app) layoutLine(fields ...segment) {
	if app.lineOffset >= len(app.screen) {
		return
	}
	nChars := len(app.screen[app.lineOffset])
	line := layout(nChars, fields...)
	for i := 0; i < nChars; i++ {
		app.screen[app.lineOffset][i] = line[i]
	}
	app.lineOffset++
}

func layout(width int, fields ...segment) []ui.Char {
	if len(fields) == 0 || len(fields) > width {
		return nil
	}

	sizes := make([]int, len(fields))
	layoutWidth := 0
	for i := range fields {
		layoutWidth += fields[i].getSize()
		sizes[i] = fields[i].getSize()
	}
	for layoutWidth < width {
		shortestFixedField, shortestFixedFieldIdx := math.MaxInt, -1
		shortestFlexField, shortestFlexFieldIdx := math.MaxInt, -1
		for j := range fields {
			if fields[j].getFlex() {
				if shortestFlexField > sizes[j] {
					shortestFlexField = sizes[j]
					shortestFlexFieldIdx = j
				}
			} else {
				if shortestFixedField > sizes[j] {
					shortestFixedField = sizes[j]
					shortestFixedFieldIdx = j
				}
			}
		}
		if shortestFlexFieldIdx != -1 {
			sizes[shortestFlexFieldIdx]++
		} else {
			sizes[shortestFixedFieldIdx]++
		}
		layoutWidth++
	}
	for layoutWidth > width {
		longestFixedField, longestFixedFieldIdx := 0, -1
		longestFlexField, longestFlexFieldIdx := 0, -1
		for j := range fields {
			if fields[j].getFlex() {
				if longestFlexField < sizes[j] {
					longestFlexField = sizes[j]
					longestFlexFieldIdx = j
				}
			} else {
				if longestFixedField < sizes[j] {
					longestFixedField = sizes[j]
					longestFixedFieldIdx = j
				}
			}
		}

		if longestFlexFieldIdx != -1 && sizes[longestFlexFieldIdx] > 1 {
			sizes[longestFlexFieldIdx]--
		} else if longestFixedFieldIdx == -1 {
			return nil
		} else {
			if sizes[longestFixedFieldIdx] > 1 {
				sizes[longestFixedFieldIdx]--
			}
		}
		layoutWidth--
	}
	result := []ui.Char{}
	for i := range fields {
		result = append(result, fields[i].render(sizes[i])...)
	}
	return result
}

type segment interface {
	getSize() int
	getFlex() bool
	render(width int) []ui.Char
}

type text struct {
	text  string
	style ui.Style
	size  int
	align alignment
	flex  bool
}

func (t text) getSize() int {
	if t.size == 0 {
		return ansi.PrintableRuneWidth(t.text)
	}
	return t.size
}

func (t text) getFlex() bool {
	return t.flex
}

func (t text) render(width int) []ui.Char {
	if width < 1 {
		return nil
	}
	runes := []rune(t.text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
	}

	diff := width - len(runes)
	idx := 0
	result := make([]ui.Char, width)
	if diff > 0 && t.align == right {
		for i := 0; i < diff; i++ {
			result[idx] = ui.Char{Rune: ' ', Style: t.style}
			idx++
		}
	}

	for i := range runes {
		result[idx] = ui.Char{Rune: runes[i], Style: t.style}
		idx++
	}

	if diff > 0 && t.align == left {
		for i := 0; i < diff; i++ {
			result[idx] = ui.Char{Rune: ' ', Style: t.style}
			idx++
		}
	}
	for ; idx < width; idx++ {
		result[idx] = ui.Char{Rune: ' ', Style: t.style}
	}

	return result
}

type progressBar struct {
	value float64
	style ui.Style
}

func (pb progressBar) getSize() int {
	return 0
}

func (pb progressBar) getFlex() bool {
	return true
}

func (pb progressBar) render(width int) []ui.Char {
	result := make([]ui.Char, width)
	progress := int(math.Round(float64(width*8) * pb.value))
	idx := 0
	for ; idx < progress/8; idx++ {
		result[idx] = ui.Char{Rune: '█', Style: pb.style}
	}
	if progress%8 > 0 {
		result[idx] = ui.Char{Rune: []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8], Style: pb.style}
		idx++
	}
	for ; idx < width; idx++ {
		result[idx] = ui.Char{Rune: ' ', Style: pb.style}
	}
	return result
}
