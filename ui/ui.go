package ui

import (
	"arch/files"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"strings"
	"time"

	"github.com/muesli/ansi"
)

type Renderer interface {
	PollEvent() any
	Render(screen Screen)
	Exit()
}

type MouseEvent struct {
	Col, Line int
}

type KeyEvent struct {
	Name string
	Rune rune
}

type ResizeEvent struct {
	Width, Height int
}

type Screen [][]Char

type Char struct {
	Style Style
	Rune  rune
}

type Style int

const (
	StyleDefault Style = iota
	StyleHeader
	StyleAppTitle
	StyleArchiveName
	StyleWhite
	StyleWhiteBold
	StyleProgressBar
)

type ui struct {
	paths       []string
	fs          files.FS
	renderer    Renderer
	screen      Screen
	locations   []location
	quit        bool
	lineOffet   int
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
	size    int
	modTime time.Time
	hash    string
}

func Run(paths []string, fs files.FS, renderer Renderer) {
	ui := &ui{
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
			tcellChan <- ui.renderer.PollEvent()
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

	for !ui.quit {
		select {
		case event := <-fsInput:
			ui.handleFsEvent(event)
		case event := <-tcellChan:
			ui.handleUiEvent(event)
		}
		ui.render()
	}
	ui.fs.Stop()
	ui.renderer.Exit()
}

func (ui *ui) analize() {
	ui.archives = make([]folder, len(ui.paths))
	for i := range ui.scanResults {
		archive := &ui.archives[i]
		archive.name = ui.paths[i]
		archive.subFolders = map[string]folder{}
		archive.files = map[string]file{}
		for _, info := range ui.scanResults[i].Files {
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
		printArchive(archive, "", "")
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

func (ui *ui) handleFsEvent(event any) {
	switch event := event.(type) {
	case *files.ScanState:
		for i := range ui.paths {
			if ui.paths[i] == event.Archive {
				ui.scanStates[i] = event
				break
			}
		}

	case *files.ArchiveInfo:
		log.Println("ArchiveInfo", event.Archive)
		for i := range ui.paths {
			if ui.paths[i] == event.Archive {
				ui.scanStates[i] = nil
				ui.scanResults[i] = event
				break
			}
		}
		doneScanning := true
		for i := range ui.paths {
			if ui.scanResults[i] == nil {
				doneScanning = false
				break
			}
		}
		if doneScanning {
			ui.analize()
		}

	default:
		log.Printf("### unhandled event %#v", event)
	}
}

func (ui *ui) handleUiEvent(event any) {
	switch ev := event.(type) {
	case ResizeEvent:
		ui.screen = make([][]Char, ev.Height)
		for line := range ui.screen {
			ui.screen[line] = make([]Char, ev.Width)
		}
		log.Printf("EventResize: cols=%d lines=%d", ev.Width, ev.Height)
	case KeyEvent:
		log.Printf("EventKey: name=%s '%c'", ev.Name, ev.Rune)
		if ev.Name == "Ctrl+C" {
			ui.quit = true
		}
		r := ev.Rune
		if r >= '1' && r <= '9' {
			idx := int(r - '1')
			if idx < len(ui.paths) {
				ui.ArchiveIdx = idx
			}
		}

	case MouseEvent:
		log.Printf("EventMouse: [%d:%d]", ev.Col, ev.Line)
	default:
	}
}

func (ui *ui) render() {
	ui.clear()
	ui.drawTitle()
	ui.drawScanStats()
	ui.drawArchive()
	ui.renderer.Render(ui.screen)
}

func (ui *ui) clear() {
	for line := range ui.screen {
		for col := range ui.screen[line] {
			ui.screen[line][col] = Char{Rune: ' ', Style: StyleWhite}
		}
	}
	ui.lineOffet = 0
}

func (ui *ui) drawTitle() {
	ui.layoutLine(
		field{text: " АРХИВАТОР ", style: StyleAppTitle},
		field{text: ui.archiveName(), style: StyleArchiveName, flex: true},
	)
}

func (ui *ui) archiveName() string {
	if ui.archives == nil {
		return ""
	}
	log.Println("### Archive", ui.archives[ui.ArchiveIdx].name)
	return ui.archives[ui.ArchiveIdx].name
}

func (ui *ui) drawScanStats() {
	if ui.scanStates == nil {
		return
	}
	for i, state := range ui.scanStates {
		if ui.scanStates[i] == nil {
			continue
		}

		ui.drawFormLine(" Архив                      ", state.Archive)
		ui.drawFormLine(" Каталог                    ", filepath.Dir(state.Name))
		ui.drawFormLine(" Документ                   ", filepath.Base(state.Name))
		ui.drawFormLine(" Ожидаемое Время Завершения ", time.Now().Add(state.Remaining).Format(time.TimeOnly))
		ui.drawFormLine(" Время До Завершения        ", state.Remaining.Truncate(time.Second).String())
		ui.layoutLine(
			field{text: " Общий Прогресс             ", style: StyleWhite},
			field{text: progressBar(len(ui.screen[ui.lineOffet])-30, state.Progress), style: StyleProgressBar, flex: true},
		)
		ui.lineOffet++
	}
}

func (ui *ui) drawFormLine(name, value string) {
	ui.layoutLine(
		field{text: name, style: StyleWhite},
		field{text: value, style: StyleWhiteBold, flex: true},
	)
}

func (ui *ui) drawArchive() {
	if ui.archives == nil {
		return
	}
	// ui.text(11, 0, ui.width-11, styleArchiveName, ui.paths[ui.ArchiveIdx])
	// archive := ui.archives[ui.ArchiveIdx]
	// location := ui.locations[ui.ArchiveIdx]
	// for _, dir := range location.path {
	// 	archive = archive.subFolders[dir]
	// }
	// subFolders := make([]folder, 0, len(archive.subFolders))
	// for _, folder := range archive.subFolders {
	// 	subFolders = append(subFolders, folder)
	// }
	// sort.Slice(subFolders, func(i, j int) bool {
	// 	return subFolders[i].name < subFolders[j].name
	// })
	// w := ui.width - 18
	// for _, subFolder := range subFolders {
	// 	ui.text(1, ui.lineOffet, 3, styleWhiteBold, "D:")
	// 	ui.text(4, ui.lineOffet, w-4, styleWhiteBold, subFolder.name)
	// 	ui.text(ui.width-18, ui.lineOffet, 18, styleWhiteBold, formatSize(subFolder.size))
	// 	if ui.lineOffet >= ui.height-2 {
	// 		break
	// 	}
	// 	ui.lineOffet++
	// }
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

type field struct {
	text  string
	style Style
	size  int
	align alignment
	flex  bool
}

func (ui *ui) layoutLine(fields ...field) {
	ui.layout(0, ui.lineOffet, len(ui.screen[ui.lineOffet]), fields)
	ui.lineOffet++
}

func (ui *ui) layout(col, line, width int, fields []field) {
	if len(fields) == 0 || len(fields) > width {
		return
	}
	layoutWidth := 0
	for i := range fields {
		if fields[i].size == 0 {
			fields[i].size = ansi.PrintableRuneWidth(fields[i].text)
		}
		layoutWidth += fields[i].size
	}
	for layoutWidth < width {
		shortestFixedField, shortestFixedFieldIdx := math.MaxInt, -1
		shortestFlexField, shortestFlexFieldIdx := math.MaxInt, -1
		for j := range fields {
			if fields[j].flex {
				if shortestFlexField > fields[j].size {
					shortestFlexField = fields[j].size
					shortestFlexFieldIdx = j
				}
			} else {
				if shortestFixedField > fields[j].size {
					shortestFixedField = fields[j].size
					shortestFixedFieldIdx = j
				}
			}
		}
		if shortestFlexFieldIdx != -1 {
			fields[shortestFlexFieldIdx].size++
		} else {
			fields[shortestFixedFieldIdx].size++
		}
		layoutWidth++
	}
	for layoutWidth > width {
		longestFixedField, longestFixedFieldIdx := 0, -1
		longestFlexField, longestFlexFieldIdx := 0, -1
		for j := range fields {
			if fields[j].flex {
				if longestFlexField < fields[j].size {
					longestFlexField = fields[j].size
					longestFlexFieldIdx = j
				}
			} else {
				if longestFixedField < fields[j].size {
					longestFixedField = fields[j].size
					longestFixedFieldIdx = j
				}
			}
		}

		if longestFlexFieldIdx != -1 && fields[longestFlexFieldIdx].size > 1 {
			fields[longestFlexFieldIdx].size--
		} else if longestFixedFieldIdx == -1 {
			return
		} else {
			if fields[longestFixedFieldIdx].size > 1 {
				fields[longestFixedFieldIdx].size--
			}
		}
		layoutWidth--
	}
	offset := 0
	for i := range fields {
		ui.text(col+offset, line, fields[i])
		offset += fields[i].size
	}
}

func (ui *ui) text(col, line int, field field) {
	if field.size < 1 {
		return
	}
	runes := []rune(field.text)
	if len(runes) > field.size {
		runes = append(runes[:field.size-1], '…')
	}

	diff := field.size - len(field.text)
	offset := 0
	if diff > 0 && field.align == right {
		for i := 0; i < diff; i++ {
			ui.screen[line][col+offset] = Char{Rune: ' ', Style: field.style}
			offset++
		}
	}

	for i := range runes {
		ui.screen[line][col+offset] = Char{Rune: runes[i], Style: field.style}
		offset++
	}

	if diff > 0 && field.align == left {
		for i := 0; i < diff; i++ {
			ui.screen[line][col+offset] = Char{Rune: ' ', Style: field.style}
			offset++
		}
	}
}

func progressBar(barWidth int, value float64) string {
	builder := strings.Builder{}
	progress := int(math.Round(float64(barWidth*8) * value))
	builder.WriteString(strings.Repeat("█", progress/8))
	if progress%8 > 0 {
		builder.WriteRune([]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8])
	}
	str := builder.String()
	length := ansi.PrintableRuneWidth(str)
	return str + strings.Repeat(" ", barWidth-length)
}
