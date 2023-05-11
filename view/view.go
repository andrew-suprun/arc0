package view

import (
	"arch/files"
	"arch/ui"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

type Model struct {
	ScanStates []*files.ScanState
	Locations  []Location
}

type File struct {
	Info   *files.FileInfo
	Kind   FileKind
	Status FileStatus
	Name   string
	Size   int
	Files  []*File
}

type Location struct {
	File       *File
	Selected   *File
	LineOffset int
}

type FileKind int

const (
	RegularFile FileKind = iota
	Folder
)

type FileStatus int

const (
	Identical FileStatus = iota
	SourceOnly
	ExtraCopy
	CopyOnly
	Discrepancy // расхождение
)

var (
	DefaultStyle       = ui.Style{FG: 231, BG: 17}
	styleAppTitle      = ui.Style{FG: 226, BG: 0, Bold: true, Italic: true}
	styleProgressBar   = ui.Style{FG: 231, BG: 19}
	styleArchiveHeader = ui.Style{FG: 231, BG: 8, Bold: true}
)

func styleFile(status FileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 17}
	if selected {
		result.Reverse = true
	}
	return result
}

func styleFolder(status FileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 18, Bold: true, Italic: true}
	if selected {
		result.Reverse = true
	}
	return result
}

func statusColor(status FileStatus) int {
	switch status {
	case Identical:
		return 250
	case SourceOnly:
		return 82
	case ExtraCopy:
		return 226
	case CopyOnly:
		return 214
	case Discrepancy:
		return 196
	}
	return 231
}

func (s FileStatus) String() string {
	switch s {
	case Identical:
		return "identical"
	case SourceOnly:
		return "sourceOnly"
	case CopyOnly:
		return "copyOnly"
	case ExtraCopy:
		return "extraCopy"
	case Discrepancy:
		return "discrepancy"
	}
	return "UNDEFINED"
}

func (s FileStatus) Merge(other FileStatus) FileStatus {
	if s > other {
		return s
	}
	return other
}

func (m Model) View() ui.Widget {
	return ui.Styled(DefaultStyle,
		ui.Column(ui.Flex(0),
			m.title(),
			m.scanStats(),
			m.treeView(),
		))
}

func (m Model) title() ui.Widget {
	return ui.Row(
		ui.Styled(styleAppTitle, ui.Text(" АРХИВАТОР", 4, 1)),
	)
}

func (m Model) scanStats() ui.Widget {
	forms := []ui.Widget{}
	for i := range m.ScanStates {
		if m.ScanStates[i] != nil {
			forms = append(forms, scanStatsForm(m.ScanStates[i]))
		}
	}
	forms = append(forms, ui.Spacer{})
	return ui.Column(0, forms...)
}

func scanStatsForm(state *files.ScanState) ui.Widget {
	return ui.Column(ui.Flex(0),
		ui.Row(ui.Text(" Архив                      ", 28, 0), ui.Text(state.Archive, 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Каталог                    ", 28, 0), ui.Text(filepath.Dir(state.Name), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Документ                   ", 28, 0), ui.Text(filepath.Base(state.Name), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Ожидаемое Время Завершения ", 28, 0), ui.Text(time.Now().Add(state.Remaining).Format(time.TimeOnly), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Время До Завершения        ", 28, 0), ui.Text(state.Remaining.Truncate(time.Second).String(), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Общий Прогресс             ", 28, 0), ui.Styled(styleProgressBar, ui.ProgressBar(state.Progress, 4, 1)), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text("", 0, 1)),
	)
}

func (m Model) treeView() ui.Widget {
	if m.Locations == nil {
		return ui.NullWidget{}
	}

	return ui.Column(1,
		ui.Styled(styleArchiveHeader,
			ui.Row(ui.Text(" Статус", 7, 0), ui.Text("  Документ", 21, 1), ui.Text(" Время Изменения", 21, 0), ui.Text("            Размер ", 19, 0)),
		),
		ui.Sized(ui.MakeConstraints(0, 1, 0, 1),
			func(width ui.W, height ui.H) ui.Widget {
				location := m.Locations[len(m.Locations)-1]
				if location.LineOffset > len(location.File.Files)-int(height) {
					location.LineOffset = len(location.File.Files) - int(height)
				}
				if location.LineOffset < 0 {
					location.LineOffset = 0
				}
				rows := make([]ui.Widget, height)
				i := 0
				var file *File
				for i, file = range location.File.Files[location.LineOffset:] {
					if i >= len(rows) {
						break
					}
					if file.Kind == RegularFile {
						rows[i] = ui.Styled(styleFile(file.Status, location.Selected == file),
							ui.Row(
								ui.Text(file.Status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.Name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text(file.Info.ModTime.Format(time.DateTime), 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.Size), 18, 0),
							),
						)
					} else {
						rows[i] = ui.Styled(styleFolder(file.Status, location.Selected == file),
							ui.Row(
								ui.Text(file.Status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.Name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text("<Каталог>", 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.Size), 18, 0),
							),
						)
					}
				}
				for i++; i < int(height); i++ {
					rows[i] = ui.Text("", 0, 1)
				}
				return ui.Column(0, rows...)
			},
		),
	)
}

func formatSize(size int) string {
	str := fmt.Sprintf("%13d ", size)
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
