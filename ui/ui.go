package ui

import (
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
	scanStats    []*scanStats
	screenHeight int
	screenWidth  int
	outChan      chan<- any
}

type scanStats struct {
	base            string
	path            string
	start           time.Time
	eta             time.Time
	remaining       time.Duration
	fileProgress    float64
	overallProgress float64
}

func Run(inChan <-chan any, outChan chan<- any) {
	output := termenv.NewOutput(os.Stdout)
	fg := output.ForegroundColor()
	bg := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetForegroundColor(fg)
		defer output.SetBackgroundColor(bg)
	}()

	p := tea.NewProgram(&model{outChan: outChan}, tea.WithAltScreen(), tea.WithMouseCellMotion())

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

	case msg.CmdScan:
		m.scanStats = append(m.scanStats, &scanStats{base: event.Base})
		return m, nil

	case msg.ScanStat:
		return m.scanStat(event)

	case msg.ScanDone:
		return m.scanDone(event)

	case msg.Analysis:
		// Оригинал
		// Копия 12
		return m.analysis(event)

	case msg.QuitApp:
		return m, tea.Quit
	}

	log.Panicf("### ui.Update received unhandled message: %#v", event)
	return m, nil
}

var nilTime time.Time

func (m *model) scanStat(stat msg.ScanStat) (tea.Model, tea.Cmd) {
	var newStat *scanStats
	for i := range m.scanStats {
		if stat.Base == m.scanStats[i].base {
			newStat = m.scanStats[i]
		}
	}
	if newStat.start == nilTime {
		newStat.start = time.Now()
	}
	newStat.path = stat.Path
	newStat.fileProgress = float64(stat.Hashed) / float64(stat.Size)
	etaProgress := float64(stat.TotalToHash) / float64(stat.TotalHashed)
	overallHashed := stat.TotalSize - stat.TotalToHash + stat.TotalHashed
	newStat.overallProgress = float64(overallHashed) / float64(stat.TotalSize)
	dur := time.Since(newStat.start)
	newStat.eta = newStat.start.Add(time.Duration(float64(dur) * etaProgress))
	newStat.remaining = time.Until(newStat.eta)

	return m, nil
}

func (m *model) scanDone(done msg.ScanDone) (tea.Model, tea.Cmd) {
	for i := range m.scanStats {
		if done.Base == m.scanStats[i].base {
			m.scanStats = append(m.scanStats[:i], m.scanStats[i+1:]...)
			break
		}
	}
	return m, nil
}

func (m *model) analysis(done msg.Analysis) (tea.Model, tea.Cmd) {
	return m, nil
}

func (m *model) View() string {
	style := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffffff")).Bold(true)
	if m.screenWidth < 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, stat := range m.scanStats {
		barWidth := m.screenWidth - 29

		builder.WriteString(header("Архив "+stat.base, m.screenWidth))
		builder.WriteString("\n Сканируется Файл           ")
		builder.WriteString(style.Render(stat.path))
		// builder.WriteString("\n Прогресс Файла             ")
		// builder.WriteString(progressBar(barWidth, stat.fileProgress))
		builder.WriteString("\n Ожидаемое Время Завершения ")
		builder.WriteString(style.Render(stat.eta.Format(time.TimeOnly)))
		builder.WriteString("\n Время До Завершения        ")
		builder.WriteString(style.Render(stat.remaining.Truncate(time.Second).String()))
		builder.WriteString("\n Общий Прогресс             ")
		builder.WriteString(progressBar(barWidth, stat.overallProgress))
		builder.WriteString("\n\n")
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

func header(text string, width int) string {
	if width <= 12 {
		return text
	}
	runes := []rune(text)
	if len(runes) > width-10 {
		runes = append(runes[:width-11], '…')
	}

	title := lipgloss.Color("#ff7f7f")

	var style = lipgloss.NewStyle().Foreground(title).Bold(true)
	out := style.Render(string(runes))
	length := ansi.PrintableRuneWidth(out)
	left := (width - length - 2) / 2
	right := width - length - 2 - left

	text = fmt.Sprintf("%s %s %s", strings.Repeat("━", left), out, strings.Repeat("━", right))
	return text
}
