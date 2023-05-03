package view

import (
	"arch/files"
	"arch/ui"
	"path/filepath"
	"time"
)

type Model struct {
	ScanStates []*files.ScanState
	Root       *File
	Location   Location
}

type File struct {
	Parent     *File
	Info       *files.FileInfo
	Kind       FileKind
	Status     FileStatus
	Name       string
	Size       int
	SubFolders []*File
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
			ui.VSpacer{},
		))
}

func (m Model) title() ui.Widget {
	return ui.Row(
		ui.Styled(ui.StyleAppTitle, ui.FlexText(" АРХИВАТОР", 1)),
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
		ui.Row(ui.Text(" Архив                      "), ui.FlexText(state.Archive, 1), ui.Text(" ")),
		ui.Row(ui.Text(" Каталог                    "), ui.FlexText(filepath.Dir(state.Name), 1), ui.Text(" ")),
		ui.Row(ui.Text(" Документ                   "), ui.FlexText(filepath.Base(state.Name), 1), ui.Text(" ")),
		ui.Row(ui.Text(" Ожидаемое Время Завершения "), ui.FlexText(time.Now().Add(state.Remaining).Format(time.TimeOnly), 1), ui.Text(" ")),
		ui.Row(ui.Text(" Время До Завершения        "), ui.FlexText(state.Remaining.Truncate(time.Second).String(), 1), ui.Text(" ")),
		ui.Row(ui.Text(" Общий Прогресс             "), ui.Styled(ui.StyleProgressBar, ui.ProgressBar(state.Progress, 4, 1)), ui.Text(" ")),
		ui.Row(ui.FlexText("", 1)),
	)
}
