package ui

import (
	"fmt"
	"log"
	"math"
	"os"
	"scanner/fs"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/termenv"
)

func Run(in <-chan any, out chan<- any) {
	output := termenv.NewOutput(os.Stdout)
	bc := output.BackgroundColor()
	defer func() {
		output := termenv.NewOutput(os.Stdout)
		defer output.SetBackgroundColor(bc)
	}()

	p := tea.NewProgram(&model{}, tea.WithAltScreen(), tea.WithMouseCellMotion())

	go func() {
		for {
			msg := <-in
			p.Send(msg)
		}
	}()
	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}

}

func (m *model) Init() tea.Cmd {
	// p := termenv.ColorProfile()
	// output := termenv.NewOutput(os.Stdout)
	// output.SetBackgroundColor(p.FromColor(color.RGBA{0, 16, 64, 255}))
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	log.Printf("msg: %#v", msg)
	switch msg := msg.(type) {
	case tea.KeyMsg:
		log.Printf("key: %v", msg)
		if s := msg.String(); s == "ctrl+c" || s == "q" || s == "esc" {
			return m, tea.Quit
		}
		return m, nil

	case tea.MouseMsg:
		log.Printf("mouse: %#v", msg)
		return m, nil

	case tea.WindowSizeMsg:
		log.Printf("size: %#v", msg)
		m.screenHeight = msg.Height
		m.screenWidth = msg.Width
		return m, nil
	case fs.ScanStat:
		return m.fileStat(msg)
	}

	log.Panicf("### received unhandled message: %#v", msg)
	return m, nil
}

func (m *model) fileStat(stat fs.ScanStat) (tea.Model, tea.Cmd) {
	log.Printf("file stat: %#v", stat)
	i := -1
	var newStat *scanStats
	for i = range m.scanStats {
		if stat.Base == m.scanStats[i].base {
			newStat = m.scanStats[i]
		}
	}
	if newStat == nil {
		newStat = &scanStats{base: stat.Base, path: stat.Path, start: time.Now()}
		m.scanStats = append(m.scanStats, newStat)
	}
	newStat.path = stat.Path
	newStat.fileProgress = float64(stat.Hashed) / float64(stat.Size)
	etaProgress := float64(stat.TotalHashed) / float64(stat.TotalToHash)
	overallHashed := stat.TotalSize - stat.TotalToHash + stat.TotalHashed
	newStat.overallProgress = float64(overallHashed) / float64(stat.TotalSize)
	dur := time.Since(newStat.start)
	eta := newStat.start.Add(time.Duration(float64(dur) / etaProgress))
	newStat.remaining = time.Until(eta)

	return m, nil
}

func (m *model) View() string {
	builder := strings.Builder{}
	for _, stat := range m.scanStats {
		barWidth := m.screenWidth - 37

		builder.WriteString("Сканнирование ")
		builder.WriteString(stat.base)
		builder.WriteString("\n Имя Файла                  ")
		builder.WriteString(stat.path)
		builder.WriteString("\n Ожидаемое Время Завершения ")
		builder.WriteString(stat.eta.Format(time.TimeOnly))
		builder.WriteString("\n Время До Завершения        ")
		builder.WriteString(stat.remaining.Truncate(time.Second).String())
		builder.WriteString("\n Прогресс Файла             ")
		builder.WriteString(fmt.Sprintf("%6.2f%% ", stat.fileProgress*100))
		builder.WriteString(progressBar(barWidth, stat.fileProgress))
		builder.WriteString("\n Общий Прогресс             ")
		builder.WriteString(fmt.Sprintf("%6.2f%% ", stat.overallProgress*100))
		builder.WriteString(progressBar(barWidth, stat.overallProgress))
		builder.WriteString("\n")
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
