package ui

import (
	"arch/files"
	"fmt"
	"log"
	"math"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/muesli/ansi"
)

type Renderer interface {
	PollEvent() any
	Render(screen Screen)
	Sync()
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
	Rune  rune
	Style Style
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
	StyleArchiveHeader
)

type ui struct {
	paths       []string
	fs          files.FS
	renderer    Renderer
	screen      Screen
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
		ui.renderer.Sync()
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
	ui.lineOffset = 0
}

func (ui *ui) drawTitle() {
	ui.layoutLine(
		text{text: " АРХИВАТОР ", style: StyleAppTitle},
		text{text: ui.archiveName(), style: StyleArchiveName, flex: true},
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
			text{text: " Общий Прогресс             ", style: StyleWhite},
			progressBar{value: state.Progress, style: StyleProgressBar},
			text{text: " ", style: StyleWhite},
		)
		ui.lineOffset++
	}
}

func (ui *ui) drawFormLine(name, value string) {
	ui.layoutLine(
		text{text: name, style: StyleWhite},
		text{text: value, style: StyleWhiteBold, flex: true},
		text{text: " ", style: StyleWhite},
	)
}

// Статус
// Документ
// Время Изменения
// Размер

func (ui *ui) drawArchive() {
	if ui.archives == nil {
		return
	}
	ui.layoutLine(
		text{text: " Статус ", size: 8, style: StyleArchiveHeader},
		text{text: "Документ ", flex: true, style: StyleArchiveHeader},
		text{text: "Время Изменения ", size: 19, style: StyleArchiveHeader},
		text{text: "Размер", size: 17, align: right, style: StyleArchiveHeader},
	)
	archive := ui.archives[ui.ArchiveIdx]
	location := ui.locations[ui.ArchiveIdx]
	for _, dir := range location.path {
		archive = archive.subFolders[dir]
	}
	subFolders := make([]folder, 0, len(archive.subFolders))
	for name, folder := range archive.subFolders {
		folder.name = name
		subFolders = append(subFolders, folder)
	}
	sort.Slice(subFolders, func(i, j int) bool {
		return subFolders[i].name < subFolders[j].name
	})
	for _, subFolder := range subFolders {
		ui.layoutLine(
			text{text: "", size: 8, style: StyleWhite},
			text{text: subFolder.name, flex: true, style: StyleWhite},
			text{text: "Каталог", size: 19, style: StyleWhite},
			text{text: formatSize(subFolder.size), size: 17, align: right, style: StyleWhite},
		)

		if ui.lineOffset >= len(ui.screen)-2 {
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

func (ui *ui) layoutLine(fields ...segment) {
	if ui.lineOffset >= len(ui.screen) {
		return
	}
	nChars := len(ui.screen[ui.lineOffset])
	line := layout(nChars, fields...)
	for i := 0; i < nChars; i++ {
		ui.screen[ui.lineOffset][i] = line[i]
	}
	ui.lineOffset++
}

func layout(width int, fields ...segment) []Char {
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
	result := []Char{}
	for i := range fields {
		result = append(result, fields[i].render(sizes[i])...)
	}
	return result
}

type segment interface {
	getSize() int
	getFlex() bool
	render(width int) []Char
}

type text struct {
	text  string
	style Style
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

func (t text) render(width int) []Char {
	if width < 1 {
		return nil
	}
	runes := []rune(t.text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
	}

	diff := width - len(runes)
	idx := 0
	result := make([]Char, width)
	if diff > 0 && t.align == right {
		for i := 0; i < diff; i++ {
			result[idx] = Char{Rune: ' ', Style: t.style}
			idx++
		}
	}

	for i := range runes {
		result[idx] = Char{Rune: runes[i], Style: t.style}
		idx++
	}

	if diff > 0 && t.align == left {
		for i := 0; i < diff; i++ {
			result[idx] = Char{Rune: ' ', Style: t.style}
			idx++
		}
	}
	for ; idx < width; idx++ {
		result[idx] = Char{Rune: ' ', Style: t.style}
	}

	return result
}

type progressBar struct {
	value float64
	style Style
}

func (pb progressBar) getSize() int {
	return 0
}

func (pb progressBar) getFlex() bool {
	return true
}

func (pb progressBar) render(width int) []Char {
	result := make([]Char, width)
	progress := int(math.Round(float64(width*8) * pb.value))
	idx := 0
	for ; idx < progress/8; idx++ {
		result[idx] = Char{Rune: '█', Style: pb.style}
	}
	if progress%8 > 0 {
		result[idx] = Char{Rune: []rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8], Style: pb.style}
		idx++
	}
	for ; idx < width; idx++ {
		result[idx] = Char{Rune: ' ', Style: pb.style}
	}
	return result
}
