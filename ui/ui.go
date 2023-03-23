package ui

import (
	"arch/files"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/muesli/ansi"
)

type UI interface {
	Run()
}

type ui struct {
	paths       []string
	fs          files.FS
	screen      tcell.Screen
	scanStates  []files.ScanState
	scanResults []*files.ArchiveInfo
	scanStarted time.Time
	archives    []folder
	archiveIdx  int
	locations   []location
	quit        bool

	width     int
	height    int
	lineOffet int
}

type folder struct {
	size       int
	subFolders map[string]folder
	files      map[string]file
}

type file struct {
	size    int
	modTime time.Time
	hash    string
}

type location struct {
	path       []string
	file       string
	lineOffset int
}

func NewUi(paths []string, fs files.FS) UI {
	ui := &ui{
		paths:       paths,
		fs:          fs,
		scanStates:  make([]files.ScanState, len(paths)),
		scanResults: make([]*files.ArchiveInfo, len(paths)),
		locations:   make([]location, len(paths)),
	}
	return ui
}

var (
	defStyle         = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	styleWhite       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x001040))
	styleWhiteBold   = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x001040)).Bold(true)
	styleTitle       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffff00)).Background(tcell.NewHexColor(0)).Bold(true).Italic(true)
	styleArchiveBane = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0)).Bold(true)
	styleProgressBar = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x1f1f9f))
)

var nilTime time.Time

func (ui *ui) Run() {
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	s.SetStyle(defStyle)
	s.EnableMouse()
	s.EnablePaste()

	ui.screen = s

	tcellChan := make(chan tcell.Event)

	go func() {
		for {
			ev := ui.screen.PollEvent()
			for {
				if ev, mouseEvent := ev.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
					break
				}
				ev = ui.screen.PollEvent()
			}
			tcellChan <- ev
		}
	}()

	inChan := make(chan any, 1)

	for _, path := range ui.paths {
		path := path
		scanChan := ui.fs.Scan(path)
		go func() {
			for scanEvent := range scanChan {
				if ui.scanStarted == nilTime {
					ui.scanStarted = time.Now()
				}
				select {
				case event := <-inChan:
					switch event.(type) {
					case files.ScanState:
						// Drop previous []files.ScanState event, if any
					default:
						inChan <- event
					}
				default:
				}

				inChan <- scanEvent
			}
		}()
	}

	for !ui.quit {
		select {
		case event := <-inChan:
			ui.handleExternalEvent(event)
		case event := <-tcellChan:
			ui.handleTcellEvent(event)
		}
		ui.render()
	}
	ui.fs.Stop()
	ui.screen.Fini()
}

func (ui *ui) handleExternalEvent(event any) {
	switch event := event.(type) {
	case files.ScanState:
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
				ui.scanStates[i].Name = ""
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

func (ui *ui) handleTcellEvent(event tcell.Event) {
	switch ev := event.(type) {
	case *tcell.EventResize:
		ui.screen.Sync()
		ui.width, ui.height = ev.Size()
		log.Printf("EventResize: cols=%d lines=%d", ui.width, ui.height)
	case *tcell.EventKey:
		log.Printf("EventKey: name=%s '%c'", ev.Name(), ev.Rune())
		if ev.Name() == "Ctrl+C" {
			ui.quit = true
		}
		r := ev.Rune()
		if r >= '1' && r <= '9' {
			idx := int(r - '1')
			if idx < len(ui.paths) {
				ui.archiveIdx = idx
			}
		}

	case *tcell.EventPaste:

	case *tcell.EventMouse:
		w, h := ev.Position()
		log.Printf("EventMouse: buttons=%v mods=%v [%d:%d]", ev.Buttons(), ev.Modifiers(), w, h)
	default:
	}
}

func (ui *ui) analize() {
	ui.archives = make([]folder, len(ui.paths))
	for i := range ui.scanResults {
		archive := &ui.archives[i]
		archive.subFolders = map[string]folder{}
		archive.files = map[string]file{}
		for _, info := range ui.scanResults[i].Files {
			log.Printf(" INFO: %s [%v]", info.Name, info.Size)
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

func (ui *ui) render() {
	ui.clear()
	ui.drawTitle()
	ui.drawScanStats()
	ui.drawArchive()
	ui.screen.Show()
}

func (ui *ui) clear() {
	for line := 1; line < ui.height-1; line++ {
		ui.text(0, line, ui.width, styleWhite, "")
	}
	ui.lineOffet = 1
}

func (ui *ui) drawTitle() {
	ui.text(0, 0, ui.width, styleTitle, " АРХИВАТОР")
	ui.text(0, ui.height-1, ui.width, styleTitle, " State")
}

func (ui *ui) drawScanStats() {
	for i, state := range ui.scanStates {
		if ui.scanResults[i] != nil {
			continue
		}
		etaProgress := float64(state.TotalToHash) / float64(state.TotalHashed)
		hashed := state.TotalSize - state.TotalToHash + state.TotalHashed
		dur := time.Since(ui.scanStarted)
		eta := ui.scanStarted.Add(time.Duration(float64(dur) * etaProgress))

		valueWidth := ui.width - 30

		ui.text(1, ui.lineOffet+0, 28, styleWhite, "Архив")
		ui.text(1, ui.lineOffet+1, 28, styleWhite, "Директория")
		ui.text(1, ui.lineOffet+2, 28, styleWhite, "Файл")
		ui.text(1, ui.lineOffet+3, 28, styleWhite, "Ожидаемое Время Завершения")
		ui.text(1, ui.lineOffet+4, 28, styleWhite, "Время До Завершения")
		ui.text(1, ui.lineOffet+5, 28, styleWhite, "Общий Прогресс")
		ui.text(29, ui.lineOffet+0, valueWidth, styleWhite, state.Archive)
		ui.text(29, ui.lineOffet+1, valueWidth, styleWhite, filepath.Dir(state.Name))
		ui.text(29, ui.lineOffet+2, valueWidth, styleWhite, filepath.Base(state.Name))
		ui.text(29, ui.lineOffet+3, valueWidth, styleWhite, eta.Format(time.TimeOnly))
		ui.text(29, ui.lineOffet+4, valueWidth, styleWhite, time.Until(eta).Truncate(time.Second).String())
		ui.text(29, ui.lineOffet+5, valueWidth, styleProgressBar, progressBar(valueWidth, float64(hashed)/float64(state.TotalSize)))
		ui.lineOffet += 7
	}
}

type renderFolder struct {
	name string
	folder
}

func (ui *ui) drawArchive() {
	if ui.archives == nil {
		return
	}
	ui.text(11, 0, ui.width-11, styleArchiveBane, ui.paths[ui.archiveIdx])
	archive := ui.archives[ui.archiveIdx]
	location := ui.locations[ui.archiveIdx]
	for _, dir := range location.path {
		archive = archive.subFolders[dir]
	}
	subFolders := make([]renderFolder, 0, len(archive.subFolders))
	for name, folder := range archive.subFolders {
		subFolders = append(subFolders, renderFolder{name, folder})
	}
	sort.Slice(subFolders, func(i, j int) bool {
		return subFolders[i].name < subFolders[j].name
	})
	w := ui.width - 18
	for _, subFolder := range subFolders {
		ui.text(1, ui.lineOffet, 3, styleWhiteBold, "D:")
		ui.text(4, ui.lineOffet, w-4, styleWhiteBold, subFolder.name)
		ui.text(ui.width-18, ui.lineOffet, 18, styleWhiteBold, formatSize(subFolder.size))
		if ui.lineOffet >= ui.height-2 {
			break
		}
		ui.lineOffet++
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

func (ui *ui) text(col, line, width int, style tcell.Style, text string) {
	if width < 1 {
		return
	}
	runes := []rune(text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
	}
	for i := range runes {
		ui.screen.SetContent(col+i, line, runes[i], nil, style)
	}
	for i := len(runes); i < width; i++ {
		ui.screen.SetContent(col+i, line, ' ', nil, style)
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
