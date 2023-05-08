package view

import (
	"arch/files"
	"arch/ui"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type Model struct {
	ScanStates []*files.ScanState
	Root       *File
	Location   Location
}

type File struct {
	Parent *File
	Info   *files.FileInfo
	Kind   FileKind
	Status FileStatus
	Name   string
	Size   int
	Files  []*File
}

type Location struct {
	Folder     *File
	File       *File
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
	return ui.Styled(ui.StyleDefault,
		ui.Column(ui.Flex(0),
			m.title(),
			m.scanStats(),
			m.treeView(),
			ui.Spacer{},
		))
}

func (m Model) title() ui.Widget {
	return ui.Row(
		ui.Styled(ui.StyleAppTitle, ui.Text(" АРХИВАТОР", 4, 1)),
	)
}

func (m Model) scanStats() ui.Widget {
	forms := []ui.Widget{}
	for i := range m.ScanStates {
		if m.ScanStates[i] != nil {
			forms = append(forms, scanStatsForm(m.ScanStates[i]))
		}
	}
	return ui.Column(0, forms...)
}

func scanStatsForm(state *files.ScanState) ui.Widget {
	return ui.Column(ui.Flex(0),
		ui.Row(ui.Text(" Архив                      ", 28, 0), ui.Text(state.Archive, 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Каталог                    ", 28, 0), ui.Text(filepath.Dir(state.Name), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Документ                   ", 28, 0), ui.Text(filepath.Base(state.Name), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Ожидаемое Время Завершения ", 28, 0), ui.Text(time.Now().Add(state.Remaining).Format(time.TimeOnly), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Время До Завершения        ", 28, 0), ui.Text(state.Remaining.Truncate(time.Second).String(), 20, 1), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text(" Общий Прогресс             ", 28, 0), ui.Styled(ui.StyleProgressBar, ui.ProgressBar(state.Progress, 4, 1)), ui.Text(" ", 1, 0)),
		ui.Row(ui.Text("", 0, 1)),
	)
}

func (m Model) treeView() ui.Widget {
	if m.Root == nil {
		return ui.NullWidget{}
	}
	return ui.List{
		Header: func() ui.Widget {
			return ui.Styled(ui.StyleArchiveHeader,
				ui.Row(ui.Text(" Статус", 7, 0), ui.Text(" Документ", 21, 1), ui.Text(" Время Изменения", 21, 0), ui.Text("            Размер ", 21, 0)),
			)
		},
		Row: func(i ui.Y) ui.Widget {
			log.Println("i =", i)
			if int(i) >= len(m.Location.File.Files) {
				return ui.Row(ui.Text("", 0, 1))
			}
			file := m.Location.File.Files[i]
			log.Printf("file = %#v", file)
			if file.Kind == RegularFile {
				return ui.Styled(ui.StyleFile,
					ui.Row(
						ui.Text("", 7, 0),
						ui.Text(" ", 1, 0),
						ui.Text(file.Name, 20, 1),
						ui.Text(" ", 1, 0),
						ui.Text(file.Info.ModTime.Format(time.DateTime), 20, 0),
						ui.Text(" ", 1, 0),
						ui.Text(formatSize(file.Size), 19, 0),
						ui.Text(" ", 1, 0),
					),
				)
			}
			return ui.Styled(ui.StyleFolder,
				ui.Row(
					ui.Text("       ", 7, 0),
					ui.Text(" ", 1, 0),
					ui.Text(file.Name, 20, 1),
					ui.Text(" ", 1, 0),
					ui.Text("Каталог", 20, 0),
					ui.Text(" ", 1, 0),
					ui.Text(formatSize(file.Size), 19, 0),
					ui.Text(" ", 1, 0),
				),
			)
		},
	}
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
