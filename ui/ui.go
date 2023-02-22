package ui

import (
	"image/color"
	"log"
	"math"
	"os"
	"scanner/fs"
	"scanner/lifecycle"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func Run(toScan []string, lc *lifecycle.Lifecycle, inChan <-chan any, outChan chan<- any) {
	output := termenv.NewOutput(os.Stdout)
	fg := output.ForegroundColor()
	bg := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetForegroundColor(fg)
		defer output.SetBackgroundColor(bg)
	}()

	p := tea.NewProgram(&model{Lifecycle: lc, scanStats: stats(toScan), outChan: outChan}, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		for {
			msg := <-inChan
			p.Send(msg)
		}
	}()
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}

func stats(toScan []string) []*scanStats {
	result := make([]*scanStats, len(toScan))
	for i, base := range toScan {
		result[i] = &scanStats{base: base}
	}
	return result
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
		if s := msg.String(); s == "ctrl+c" || s == "esc" {
			m.Lifecycle.Stop()
			return m, nil
		}
		return m, nil

	case tea.MouseMsg:
		return m, nil

	case tea.WindowSizeMsg:
		m.screenHeight = msg.Height
		m.screenWidth = msg.Width
		return m, nil
	case fs.ScanStat:
		return m.scanFileStat(msg)
	case fs.ScanDone:
		return m.scanDone(msg)
	}

	log.Panicf("### received unhandled message: %#v", msg)
	return m, nil
}

var nilTime time.Time

func (m *model) scanFileStat(stat fs.ScanStat) (tea.Model, tea.Cmd) {
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

func (m *model) scanDone(done fs.ScanDone) (tea.Model, tea.Cmd) {
	for i := range m.scanStats {
		if done.Base == m.scanStats[i].base {
			m.scanStats = append(m.scanStats[:i], m.scanStats[i+1:]...)
			break
		}
	}
	return m.checkDone()
}

func (m *model) checkDone() (tea.Model, tea.Cmd) {
	if len(m.scanStats) == 0 {
		return m, tea.Quit
	}
	return m, nil
}

func (m *model) View() string {
	builder := strings.Builder{}
	for _, stat := range m.scanStats {
		barWidth := m.screenWidth - 29

		builder.WriteString(" Архив                      ")
		builder.WriteString(stat.base)
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

var runes = []rune{' ', '\u258F', '\u258E', '\u258D', '\u258C', '\u258A', '\u258A', '\u2589'}

func progressBar(barWidth int, value float64) string {
	builder := strings.Builder{}
	progress := int(math.Round(float64(barWidth*8) * value))
	builder.WriteString(strings.Repeat("\u2588", progress/8))
	builder.WriteRune(runes[progress%8])
	return builder.String()
}
