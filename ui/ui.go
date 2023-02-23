package ui

import (
	"fmt"
	"image/color"
	"log"
	"math"
	"os"
	"scanner/api"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
			msg := <-inChan
			p.Send(msg)
			if msg == tea.Quit() {
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
	output.SetForegroundColor(p.FromColor(color.RGBA{255, 255, 64, 255}))
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		s := msg.String()
		switch s {
		case "ctrl+c", "esc":
			m.outChan <- api.CmdQuit{}
			return m, nil
		}
		return m, nil

	case tea.MouseMsg:
		return m, nil

	case tea.WindowSizeMsg:
		cmd := tea.Cmd(nil)
		if m.screenWidth > msg.Width {
			cmd = tea.ClearScreen
		}
		m.screenHeight = msg.Height
		m.screenWidth = msg.Width
		return m, cmd

	case api.CmdScan:
		m.scanStats = append(m.scanStats, &scanStats{base: msg.Base})
		return m, nil

	case api.ScanStat:
		return m.scanFileStat(msg)

	case api.ScanDone:
		return m.scanDone(msg)
	}

	log.Printf("### ui.Update received unhandled message: %#v", msg)
	return m, nil
}

var nilTime time.Time

func (m *model) scanFileStat(stat api.ScanStat) (tea.Model, tea.Cmd) {
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

func (m *model) scanDone(done api.ScanDone) (tea.Model, tea.Cmd) {
	for i := range m.scanStats {
		if done.Base == m.scanStats[i].base {
			m.scanStats = append(m.scanStats[:i], m.scanStats[i+1:]...)
			break
		}
	}
	return m, nil
}

func (m *model) View() string {
	if m.screenWidth < 0 {
		return ""
	}
	builder := strings.Builder{}
	for _, stat := range m.scanStats {
		barWidth := m.screenWidth - 29

		builder.WriteString(header("Архив "+stat.base, m.screenWidth))
		builder.WriteString("\n Имя Файла                  ")
		builder.WriteString(stat.path)
		builder.WriteString("\n Ожидаемое Время Завершения ")
		builder.WriteString(stat.eta.Format(time.TimeOnly))
		builder.WriteString("\n Время До Завершения        ")
		builder.WriteString(stat.remaining.Truncate(time.Second).String())
		builder.WriteString("\n Прогресс Файла             ")
		builder.WriteString(progressBar(barWidth, stat.fileProgress))
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
	builder.WriteRune([]rune{' ', '▏', '▎', '▍', '▌', '▋', '▊', '▉'}[progress%8])
	return builder.String()
}

func header(text string, width int) string {
	log.Println("### text", text, "len", len(text), "width", width)
	if width <= 12 {
		return text
	}
	runes := []rune(text)
	if len(runes) > width-10 {
		runes = append(runes[:width-11], '…')
	}
	log.Println("### text", text, "len", len(text), "width", width)
	count := width - len(runes) - 6
	if width <= 0 {
		return text
	}
	text = fmt.Sprintf("━━━━ %s %s", string(runes), strings.Repeat("━", count))
	log.Println("### text", text, "len", len(text), "count", count)
	return text
}
