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

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/ansi"
	"github.com/muesli/termenv"
)

type model struct {
	lc           *lifecycle.Lifecycle
	outChan      chan<- any
	screenHeight int
	screenWidth  int
	scanStart    time.Time
	scanState    []msg.ScanState
}

type scanState struct {
	eta             time.Time
	remaining       time.Duration
	fileProgress    float64
	overallProgress float64
}

func Run(lc *lifecycle.Lifecycle, inChan <-chan any, outChan chan<- any) {
	output := termenv.NewOutput(os.Stdout)
	fg := output.ForegroundColor()
	bg := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetForegroundColor(fg)
		defer output.SetBackgroundColor(bg)
	}()

	p := tea.NewProgram(&model{lc: lc, outChan: outChan}, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		for {
			event := <-inChan
			p.Send(event)
			if event == tea.Quit() {
				return
			}
		}
	}()
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func (m *model) Init() tea.Cmd {
	p := termenv.ColorProfile()
	output := termenv.NewOutput(os.Stdout)
	output.SetBackgroundColor(p.FromColor(color.RGBA{0, 16, 64, 255}))
	output.SetForegroundColor(p.FromColor(color.RGBA{255, 255, 205, 255}))
	return nil
}

func (m *model) Update(event tea.Msg) (tea.Model, tea.Cmd) {
	switch event := event.(type) {
	case tea.KeyMsg:
		s := event.String()
		switch s {
		case "ctrl+c", "esc":
			m.outChan <- msg.CmdQuit{}
			return m, nil
		}
		return m, nil

	case tea.MouseMsg:
		return m, nil

	case tea.WindowSizeMsg:
		cmd := tea.Cmd(nil)
		if m.screenWidth > event.Width {
			cmd = tea.ClearScreen
		}
		m.screenHeight = event.Height
		m.screenWidth = event.Width
		return m, cmd

	case []msg.ScanState:
		return m.scanStateEvent(event)

	case msg.ArchiveInfo:
		log.Printf("ui: event=%#v\n", event)
		// Оригинал
		// Копия 12
		return m.analysis(event)

	case msg.QuitApp:
		m.lc.Stop()
		return m, tea.Quit
	}

	log.Panicf("### ui.Update received unhandled message: %#v", event)
	return m, nil
}

var nilTime time.Time

func (m *model) scanStateEvent(scanState []msg.ScanState) (tea.Model, tea.Cmd) {
	if m.scanStart == nilTime {
		m.scanStart = time.Now()
	}
	m.scanState = scanState
	return m, nil
}

func (m *model) analysis(done msg.ArchiveInfo) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *model) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
	if m.screenWidth < 0 {
		return ""
	}
	builder := strings.Builder{}
	builder.WriteString(screenTitle(m.screenWidth))
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

		barWidth := m.screenWidth - 29

		builder.WriteString("\n Архив                      ")
		builder.WriteString(style.Render(state.Base))
		builder.WriteString("\n Сканируется Файл           ")
		builder.WriteString(style.Render(state.Path))
		builder.WriteString("\n Ожидаемое Время Завершения ")
		builder.WriteString(style.Render(eta.Format(time.TimeOnly)))
		builder.WriteString("\n Время До Завершения        ")
		builder.WriteString(style.Render(remaining.Truncate(time.Second).String()))
		builder.WriteString("\n Общий Прогресс             ")
		builder.WriteString(progressBar(barWidth, overallProgress))
		builder.WriteString("\n")
	}
	return builder.String()
}

func progressBar(barWidth int, value float64) string {
	if barWidth <= 0 {
		return ""
	}
	builder := strings.Builder{}
	progress := int(math.Round(float64(barWidth*8) * value))
	builder.WriteString(strings.Repeat("█", progress/8))
	if progress%8 > 0 {
		builder.WriteRune([]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8])
	}
	style := lipgloss.NewStyle().Background(lipgloss.Color("#1f1f9f"))
	str := builder.String()
	length := ansi.PrintableRuneWidth(str)
	return style.Render(str + strings.Repeat(" ", barWidth-length))
}

func screenTitle(width int) string {
	if width == 0 {
		return ""
	}
	var style = lipgloss.NewStyle().Foreground(lipgloss.Color("#001040")).Background(lipgloss.Color("#7fff7f")).Bold(true)
	title := fmt.Sprintf(" АРХИВАТОР %s", strings.Repeat(" ", width-11))
	return style.Render(title)
}
