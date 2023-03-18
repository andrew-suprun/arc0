package ui

import (
	"arch/files"
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/muesli/ansi"
	"github.com/muesli/termenv"
)

type UI interface {
	Run()
}

type ui struct {
	paths       []string
	fs          files.FS
	screen      tcell.Screen
	scanStates  []files.ScanState
	archives    []*files.ArchiveInfo
	scanDone    []bool
	scanStarted time.Time
	quit        bool
}

type scanDone struct {
	archive string
}

func NewUi(paths []string, fs files.FS) UI {
	ui := &ui{
		paths:      paths,
		fs:         fs,
		scanStates: make([]files.ScanState, len(paths)),
		archives:   make([]*files.ArchiveInfo, len(paths)),
		scanDone:   make([]bool, len(paths)),
	}
	return ui
}

var (
	defStyle         = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	styleWhite       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x001040))
	styleTitle       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x001040)).Background(tcell.NewHexColor(0xdfdfdf)).Bold(true)
	styleProgressBar = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x1f1f9f))
)

var nilTime time.Time

func (ui *ui) Run() {
	output := termenv.NewOutput(os.Stdout)
	fg := output.ForegroundColor()
	bg := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetForegroundColor(fg)
		defer output.SetBackgroundColor(bg)
	}()
	p := termenv.ColorProfile()
	output.SetBackgroundColor(p.FromColor(color.RGBA{0, 16, 64, 255}))
	output.SetForegroundColor(p.FromColor(color.RGBA{255, 255, 205, 255}))

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
				log.Printf("scan event: %#v", scanEvent)
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
			inChan <- scanDone{archive: path}
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

	case scanDone:
		for i := range ui.paths {
			if ui.paths[i] == event.archive {
				ui.scanDone[i] = true
				break
			}
		}

	case *files.ArchiveInfo:
		for i := range ui.paths {
			if ui.paths[i] == event.Archive {
				ui.scanStates[i].Name = ""
				ui.archives[i] = event
				break
			}
		}
		doneScanning := true
		for i := range ui.paths {
			if ui.archives[i] == nil {
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
		w, h := ev.Size()
		log.Printf("EventResize: cols=%d lines=%d", w, h)
	case *tcell.EventKey:
		log.Printf("EventKey: name=%s", ev.Name())
		if ev.Name() == "Esc" {
			ui.quit = true
		}

	case *tcell.EventPaste:

	case *tcell.EventMouse:
		w, h := ev.Position()
		log.Printf("EventKey: buttons=%v mods=%v [%d:%d]", ev.Buttons(), ev.Modifiers(), w, h)
	default:
	}
}

func (ui *ui) analize() {
	log.Println("### ANALIZE")
}

func (ui *ui) render() {
	ui.screen.Clear()
	w, h := ui.screen.Size()

	line := 1
	for i, state := range ui.scanStates {
		log.Printf("render: i=%d done=%v", i, ui.scanDone[i])
		if ui.scanDone[i] {
			continue
		}
		etaProgress := float64(state.TotalToHash) / float64(state.TotalHashed)
		hashed := state.TotalSize - state.TotalToHash + state.TotalHashed
		dur := time.Since(ui.scanStarted)
		eta := ui.scanStarted.Add(time.Duration(float64(dur) * etaProgress))

		valueWidth := w - 30

		ui.text(1, line+0, 28, styleWhite, "Архив")
		ui.text(1, line+1, 28, styleWhite, "Директория")
		ui.text(1, line+2, 28, styleWhite, "Файл")
		ui.text(1, line+3, 28, styleWhite, "Ожидаемое Время Завершения")
		ui.text(1, line+4, 28, styleWhite, "Время До Завершения")
		ui.text(1, line+5, 28, styleWhite, "Общий Прогресс")
		ui.text(29, line+0, valueWidth, styleWhite, state.Archive)
		ui.text(29, line+1, valueWidth, styleWhite, state.Folder)
		ui.text(29, line+2, valueWidth, styleWhite, state.Name)
		ui.text(29, line+3, valueWidth, styleWhite, eta.Format(time.TimeOnly))
		ui.text(29, line+4, valueWidth, styleWhite, time.Until(eta).Truncate(time.Second).String())
		ui.text(29, line+5, valueWidth, styleProgressBar, progressBar(valueWidth, float64(hashed)/float64(state.TotalSize)))
		line += 7
	}

	ui.text(0, 0, w, styleTitle, " АРХИВАТОР")
	ui.text(0, h-1, w, styleTitle, " State")
	ui.screen.Show()
}

func (ui *ui) text(col, line, width int, style tcell.Style, text string) {
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
	log.Println("value", value, "bar", barWidth, "len", length)
	return str + strings.Repeat(" ", barWidth-length)
}
