package ui

import (
	"arch/lifecycle"
	"arch/msg"
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

type model struct {
	lc        *lifecycle.Lifecycle
	outChan   chan<- any
	screen    tcell.Screen
	scanStart time.Time
	scanState []msg.ScanState
}

var done = false

var (
	defStyle         = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	styleWhite       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x001040))
	styleTitle       = tcell.StyleDefault.Foreground(tcell.NewHexColor(0x001040)).Background(tcell.NewHexColor(0x7fff7f)).Bold(true)
	styleProgressBar = tcell.StyleDefault.Foreground(tcell.NewHexColor(0xffffff)).Background(tcell.NewHexColor(0x1f1f9f))
)

func Run(lc *lifecycle.Lifecycle, inChan <-chan any, outChan chan<- any) {
	output := termenv.NewOutput(os.Stdout)
	fg := output.ForegroundColor()
	bg := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetForegroundColor(fg)
		defer output.SetBackgroundColor(bg)
	}()

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

	p := termenv.ColorProfile()
	output.SetBackgroundColor(p.FromColor(color.RGBA{0, 16, 64, 255}))
	output.SetForegroundColor(p.FromColor(color.RGBA{255, 255, 205, 255}))

	m := &model{lc: lc, outChan: outChan, screen: s}

	tcellChan := make(chan tcell.Event)

	go func() {
		for {
			log.Printf("about to poll event 1")
			ev := m.screen.PollEvent()
			for {
				if ev, mouseEvent := ev.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
					break
				}
				log.Printf("about to poll event 2")
				ev = m.screen.PollEvent()
			}
			log.Printf("tcell event %#v", ev)
			tcellChan <- ev
			log.Printf("tcell sent event")
		}
	}()

	for !done {
		select {
		case event := <-inChan:
			m.handleExternalEvent(event)
		case event := <-tcellChan:
			m.handleTcellEvent(event)
		}
		m.render()
	}
	log.Println("ui: event=quit 1")
	m.lc.Stop()
	log.Println("ui: event=quit 2")
	m.screen.Fini()
	log.Println("ui: event=quit 3")
}

func (m *model) handleExternalEvent(event any) {
	switch event := event.(type) {
	case []msg.ScanState:
		m.scanStateEvent(event)

	case msg.ArchiveInfo:
		log.Printf("ui: event=%#v\n", event)
		// Оригинал
		// Копия 12
		m.analysis(event)

	case msg.QuitApp:
		done = true
	}
}

func (m *model) handleTcellEvent(event tcell.Event) {
	switch ev := event.(type) {
	case *tcell.EventResize:
		m.screen.Sync()
		w, h := ev.Size()
		log.Printf("EventResize: cols=%d lines=%d", w, h)
	case *tcell.EventKey:
		log.Printf("EventKey: name=%s", ev.Name())
		if ev.Name() == "Esc" {
			m.outChan <- msg.CmdQuit{}
		}

	case *tcell.EventPaste:

	case *tcell.EventMouse:
		w, h := ev.Position()
		log.Printf("EventKey: buttons=%v mods=%v [%d:%d]", ev.Buttons(), ev.Modifiers(), w, h)
	default:
	}
}

func (m *model) render() {
	m.screen.Clear()
	w, h := m.screen.Size()
	m.text(0, 0, w, styleTitle, " АРХИВАТОР")

	line := 1
	for _, state := range m.scanState {
		if state.Path == "" {
			continue
		}
		etaProgress := float64(state.TotalToHash) / float64(state.TotalHashed)
		overallHashed := state.TotalSize - state.TotalToHash + state.TotalHashed
		overallProgress := float64(overallHashed) / float64(state.TotalSize)
		dur := time.Since(m.scanStart)
		eta := m.scanStart.Add(time.Duration(float64(dur) * etaProgress))
		remaining := time.Until(eta)

		valueWidth := w - 29

		m.text(1, line+0, 28, styleWhite, "Архив")
		m.text(1, line+1, 28, styleWhite, "Сканируется Файл")
		m.text(1, line+2, 28, styleWhite, "Ожидаемое Время Завершения")
		m.text(1, line+3, 28, styleWhite, "Время До Завершения")
		m.text(1, line+4, 28, styleWhite, "Общий Прогресс")
		m.text(29, line+0, valueWidth, styleWhite, state.Base)
		m.text(29, line+1, valueWidth, styleWhite, state.Path)
		m.text(29, line+2, valueWidth, styleWhite, eta.Format(time.TimeOnly))
		m.text(29, line+3, valueWidth, styleWhite, remaining.Truncate(time.Second).String())
		m.text(29, line+4, valueWidth, styleProgressBar, progressBar(valueWidth, overallProgress))
		line += 6
	}

	m.text(0, h-1, w, styleTitle, " State")
	m.screen.Show()
}

func (m *model) text(col, line, width int, style tcell.Style, text string) {
	runes := []rune(text)
	if len(runes) > width {
		runes = append(runes[:width-1], '…')
	}
	for i := range runes {
		m.screen.SetContent(col+i, line, runes[i], nil, style)
	}
	for i := len(runes); i < width; i++ {
		m.screen.SetContent(col+i, line, ' ', nil, style)
	}
}

var nilTime time.Time

func (m *model) scanStateEvent(scanState []msg.ScanState) {
	if m.scanStart == nilTime {
		m.scanStart = time.Now()
	}
	m.scanState = scanState
}

func (m *model) analysis(done msg.ArchiveInfo) {
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
