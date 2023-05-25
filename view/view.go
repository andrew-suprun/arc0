package view

import (
	"arch/device"
	"arch/model"
	. "arch/ui"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	styleAppTitle      = device.Style{FG: 226, BG: 0, Flags: device.Bold + device.Italic}
	styleStatusLine    = device.Style{FG: 226, BG: 0}
	styleProgressBar   = device.Style{FG: 231, BG: 19}
	styleArchiveHeader = device.Style{FG: 231, BG: 8, Flags: device.Bold}
)

func Draw(m *model.Model) Widget {
	return Column(0,
		title(),
		scanStats(m),
		treeView(m),
		statusLine(m),
	)
}

func title() Widget {
	return Row(
		Styled(styleAppTitle, Text(" АРХИВАТОР").Flex(1)),
	)
}

func scanStats(m *model.Model) Widget {
	forms := []Widget{}
	first := true
	for i := range m.Archives {
		if m.Archives[i].ScanState != nil {
			if !first {
				forms = append(forms, Row(Text("").Flex(1).Pad('─')))
			}
			forms = append(forms, scanStatsForm(m.Archives[i].Path, m.Archives[i].ScanState))
			first = false
		}
	}
	if len(forms) == 0 {
		return NullWidget{}
	}
	forms = append(forms, Spacer{})
	return Column(1, forms...)
}

func scanStatsForm(archive string, state *model.ScanState) Widget {
	return Column(0,
		Row(Text(" Архив                       "), Text(archive).Flex(1), Text(" ")),
		Row(Text(" Каталог                     "), Text(filepath.Dir(state.Name)).Flex(1), Text(" ")),
		Row(Text(" Документ                    "), Text(filepath.Base(state.Name)).Flex(1), Text(" ")),
		Row(Text(" Ожидаемое Время Завершения  "), Text(time.Now().Add(state.Remaining).Format(time.TimeOnly)).Flex(1), Text(" ")),
		Row(Text(" Время До Завершения         "), Text(state.Remaining.Truncate(time.Second).String()).Flex(1), Text(" ")),
		Row(Text(" Общий Прогресс              "), Styled(styleProgressBar, ProgressBar(state.Progress)), Text(" ")),
	)
}

func treeView(m *model.Model) Widget {
	if len(m.Breadcrumbs) == 0 {
		return NullWidget{}
	}

	return Column(1,
		breadcrumbs(m),
		Styled(styleArchiveHeader,
			Row(
				MouseTarget(model.SortByStatus, Text(" Статус"+sortIndicator(m, model.SortByStatus)).Width(13)),
				MouseTarget(model.SortByName, Text("  Документ"+sortIndicator(m, model.SortByName)).Width(20).Flex(1)),
				MouseTarget(model.SortByTime, Text("  Время Изменения"+sortIndicator(m, model.SortByTime)).Width(19)),
				MouseTarget(model.SortBySize, Text(fmt.Sprintf("%22s", "Размер"+sortIndicator(m, model.SortBySize)+" "))),
			),
		),
		Scroll(nil, device.Constraint{Size: device.Size{Width: 0, Height: 0}, Flex: device.Flex{X: 1, Y: 1}},
			func(size device.Size) Widget {
				m.FileTreeLines = size.Height
				folder := m.CurerntFolder()
				if folder.LineOffset > len(folder.File.Files)+1-size.Height {
					folder.LineOffset = len(folder.File.Files) + 1 - size.Height
				}
				if folder.LineOffset < 0 {
					folder.LineOffset = 0
				}
				rows := []Widget{}
				i := 0
				var file *model.FileInfo
				for i, file = range folder.File.Files[folder.LineOffset:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, Styled(styleFile(file, folder.Selected == file),
						MouseTarget(model.SelectFile(file), Row(
							Text(" "+repr(file.Status)).Width(13),
							Text("  "),
							Text(displayName(file)).Width(20).Flex(1),
							Text("  "),
							Text(file.ModTime.Format(time.DateTime)),
							Text("  "),
							Text(formatSize(file.Size)).Width(18),
						)),
					))
				}
				rows = append(rows, Spacer{})
				return Column(0, rows...)
			},
		),
	)
}

func displayName(file *model.FileInfo) string {
	if file.Kind == model.FileFolder {
		return "▶ " + file.Name
	}
	return "  " + file.Name
}

func sortIndicator(m *model.Model, column model.SortColumn) string {
	if column == m.SortColumn {
		if m.SortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func breadcrumbs(m *model.Model) Widget {
	widgets := make([]Widget, 0, len(m.Breadcrumbs)*2)
	for i, folder := range m.Breadcrumbs {
		if i > 0 {
			widgets = append(widgets, Text(" / "))
		}
		widgets = append(widgets,
			MouseTarget(model.SelectFolder(folder.File),
				Styled(styleBreadcrumbs(folder.File), Text(folder.File.Name)),
			),
		)
	}
	widgets = append(widgets, Spacer{})
	return Row(widgets...)
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

func statusLine(m *model.Model) Widget {
	return Row(
		Styled(styleStatusLine, Text(" Status line will be here...").Flex(1)),
	)
}

func styleFile(file *model.FileInfo, selected bool) device.Style {
	bg, flags := byte(17), device.Flags(0)
	if file.Kind == model.FileFolder {
		bg = byte(18)
	}
	result := device.Style{FG: statusColor(file.Status), BG: bg, Flags: flags}
	if selected {
		result.Flags |= device.Reverse
	}
	return result
}

func styleBreadcrumbs(file *model.FileInfo) device.Style {
	return device.Style{FG: statusColor(file.Status), BG: 17, Flags: device.Bold + device.Italic}
}

func statusColor(status model.FileStatus) byte {
	switch status {
	case model.Identical:
		return 250
	case model.SourceOnly:
		return 82
	case model.CopyOnly:
		return 196
	}
	return 231
}

func repr(status model.FileStatus) string {
	switch status {
	case model.Identical:
		return ""
	case model.SourceOnly:
		return "Оригинал"
	case model.CopyOnly:
		return "Только Копия"
	}
	return "UNDEFINED"
}
