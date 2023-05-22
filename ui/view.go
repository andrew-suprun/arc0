package ui

import (
	"arch/device"
	"arch/files"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

var (
	defaultStyle       = device.Style{FG: 231, BG: 17}
	styleAppTitle      = device.Style{FG: 226, BG: 0, Flags: device.Bold + device.Italic}
	styleStatusLine    = device.Style{FG: 226, BG: 0}
	styleProgressBar   = device.Style{FG: 231, BG: 19}
	styleArchiveHeader = device.Style{FG: 231, BG: 8, Flags: device.Bold}
)

func (m *model) handleDeviceEvent(event device.Event) bool {
	switch event := event.(type) {
	case device.ResizeEvent:
		m.screenSize = Size(event)

	case device.KeyEvent:
		result := m.handleKeyEvent(event)
		m.makeSelectedVisible()
		return result

	case device.MouseEvent:
		m.handleMouseEvent(event)

	case device.ScrollEvent:
		if event.Direction == device.ScrollUp {
			m.currentLocation().lineOffset++
		} else {
			m.currentLocation().lineOffset--
		}

	default:
		log.Panicf("### unhandled device event %#v", event)
	}
	return true
}

func (m *model) handleKeyEvent(key device.KeyEvent) bool {
	if key.Name == "Ctrl+C" {
		return false
	}

	loc := m.currentLocation()

	switch key.Name {
	case "Enter":
		m.enter()

	case "Esc":
		m.esc()

	case "Rune[R]", "Rune[r]":
		exec.Command("open", "-R", loc.selected.path).Start()

	case "Home":
		loc.selected = loc.file.files[0]

	case "End":
		loc.selected = loc.file.files[len(loc.file.files)-1]

	case "PgUp":
		loc.lineOffset -= m.archiveViewLines
		if loc.lineOffset < 0 {
			loc.lineOffset = 0
		}
		idxSelected := 0
		foundSelected := false
		for i := 0; i < len(loc.file.files); i++ {
			if loc.file.files[i] == loc.selected {
				idxSelected = i
				foundSelected = true
				break
			}
		}
		if foundSelected {
			idxSelected -= m.archiveViewLines
			if idxSelected < 0 {
				idxSelected = 0
			}
			loc.selected = loc.file.files[idxSelected]
		}

	case "PgDn":
		loc.lineOffset += m.archiveViewLines
		if loc.lineOffset > len(loc.file.files)-m.archiveViewLines {
			loc.lineOffset = len(loc.file.files) - m.archiveViewLines
		}
		idxSelected := 0
		foundSelected := false
		for i := 0; i < len(loc.file.files); i++ {
			if loc.file.files[i] == loc.selected {
				idxSelected = i
				foundSelected = true
				break
			}
		}
		if foundSelected {
			idxSelected += m.archiveViewLines
			if idxSelected > len(loc.file.files)-1 {
				idxSelected = len(loc.file.files) - 1
			}
			loc.selected = loc.file.files[idxSelected]
		}

	case "Up":
		m.up()

	case "Down":
		m.down()
	}
	return true
}

func (m *model) handleMouseEvent(event device.MouseEvent) {
	for _, target := range m.ctx.MouseTargetAreas {
		if target.Pos.X <= event.X && target.Pos.X+target.Size.Width > event.X &&
			target.Pos.Y <= event.Y && target.Pos.Y+target.Size.Height > event.Y {

			switch cmd := target.Command.(type) {
			case selectFolder:
				for i, loc := range m.locations {
					if loc.file == cmd && i < len(m.locations) {
						m.locations = m.locations[:i+1]
						return
					}
				}
			case selectFile:
				m.currentLocation().selected = cmd
				last := m.lastMouseEvent
				if event.Time.Sub(last.Time).Seconds() < 0.5 {
					m.enter()
				}
				m.lastMouseEvent = event
			case sortColumn:
				if cmd == m.sortColumn {
					m.sortAscending[m.sortColumn] = !m.sortAscending[m.sortColumn]
				} else {
					m.sortColumn = cmd
				}
				m.sort()
			}
		}
	}
}

func styleFile(file *fileInfo, selected bool) device.Style {
	bg, flags := byte(17), device.Flags(0)
	if file.status == discrepancy {
		flags = device.Bold
	}
	if file.kind == folder {
		bg = byte(18)
	}
	result := device.Style{FG: statusColor(file.status), BG: bg, Flags: flags}
	if selected {
		result.Flags |= device.Reverse
	}
	return result
}

func styleBreadcrumbs(file *fileInfo) device.Style {
	return device.Style{FG: statusColor(file.status), BG: 17, Flags: device.Bold + device.Italic}
}

func statusColor(status fileStatus) byte {
	switch status {
	case identical:
		return 250
	case sourceOnly:
		return 82
	case extraCopy:
		return 226
	case copyOnly:
		return 214
	case discrepancy:
		return 196
	}
	return 231
}

func (s fileStatus) repr() string {
	switch s {
	case identical:
		return ""
	case sourceOnly:
		return "Оригинал"
	case copyOnly:
		return "Только Копия"
	case extraCopy:
		return "Лишняя Копия"
	case discrepancy:
		return "Расхождение"
	}
	return "UNDEFINED"
}

func (m *model) title() Widget {
	return Row(
		Styled(styleAppTitle, Text(" АРХИВАТОР").Flex(1)),
	)
}

func (m *model) statusLine() Widget {
	return Row(
		Styled(styleStatusLine, Text(" Status line will be here...").Flex(1)),
	)
}

func (m *model) scanStats() Widget {
	if m.scanStates == nil {
		return NullWidget{}
	}
	forms := []Widget{}
	first := true
	for i := range m.scanStates {
		if m.scanStates[i] != nil {
			if !first {
				forms = append(forms, Row(Text("").Flex(1).Pad('─')))
			}
			forms = append(forms, scanStatsForm(m.scanStates[i]))
			first = false
		}
	}
	forms = append(forms, Spacer{})
	return Column(1, forms...)
}

func scanStatsForm(state *files.ScanState) Widget {
	return Column(0,
		Row(Text(" Архив                       "), Text(state.Archive).Flex(1), Text(" ")),
		Row(Text(" Каталог                     "), Text(filepath.Dir(state.Name)).Flex(1), Text(" ")),
		Row(Text(" Документ                    "), Text(filepath.Base(state.Name)).Flex(1), Text(" ")),
		Row(Text(" Ожидаемое Время Завершения  "), Text(time.Now().Add(state.Remaining).Format(time.TimeOnly)).Flex(1), Text(" ")),
		Row(Text(" Время До Завершения         "), Text(state.Remaining.Truncate(time.Second).String()).Flex(1), Text(" ")),
		Row(Text(" Общий Прогресс              "), Styled(styleProgressBar, ProgressBar(state.Progress)), Text(" ")),
	)
}

func (m *model) treeView() Widget {
	if len(m.locations) == 0 {
		return NullWidget{}
	}

	return Column(1,
		m.breadcrumbs(),
		Styled(styleArchiveHeader,
			Row(
				MouseTarget(sortByStatus, Text(" Статус"+m.sortIndicator(sortByStatus)).Width(13)),
				MouseTarget(sortByName, Text("  Документ"+m.sortIndicator(sortByName)).Width(20).Flex(1)),
				MouseTarget(sortByTime, Text("  Время Изменения"+m.sortIndicator(sortByTime)).Width(19)),
				MouseTarget(sortBySize, Text(fmt.Sprintf("%22s", "Размер"+m.sortIndicator(sortBySize)+" "))),
			),
		),
		Scroll(nil, Constraint{Size: Size{Width: 0, Height: 0}, Flex: Flex{X: 1, Y: 1}},
			func(size Size) Widget {
				m.archiveViewLines = size.Height
				location := m.currentLocation()
				if location.lineOffset > len(location.file.files)+1-size.Height {
					location.lineOffset = len(location.file.files) + 1 - size.Height
				}
				if location.lineOffset < 0 {
					location.lineOffset = 0
				}
				rows := []Widget{}
				i := 0
				var file *fileInfo
				for i, file = range location.file.files[location.lineOffset:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, Styled(styleFile(file, location.selected == file),
						MouseTarget(selectFile(file), Row(
							Text(" "+file.status.repr()).Width(13),
							Text("  "),
							Text(displayName(file)).Width(20).Flex(1),
							Text("  "),
							Text(file.modTime.Format(time.DateTime)),
							Text("  "),
							Text(formatSize(file.size)).Width(18),
						)),
					))
				}
				rows = append(rows, Spacer{})
				return Column(0, rows...)
			},
		),
	)
}

func (m *model) makeSelectedVisible() {
	location := m.currentLocation()
	if location.selected == nil {
		return
	}
	idx := -1
	for i := range location.file.files {
		if location.selected == location.file.files[i] {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if location.lineOffset > idx {
			location.lineOffset = idx
		}
		if location.lineOffset < idx+1-m.archiveViewLines {
			location.lineOffset = idx + 1 - m.archiveViewLines
		}
	}
}

func displayName(file *fileInfo) string {
	if file.kind == folder {
		return "▶ " + file.name
	}
	return "  " + file.name
}

func (m *model) sortIndicator(column sortColumn) string {
	if column == m.sortColumn {
		if m.sortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func (m *model) breadcrumbs() Widget {
	widgets := make([]Widget, 0, len(m.locations)*2)
	for i, loc := range m.locations {
		if i > 0 {
			widgets = append(widgets, Text(" / "))
		}
		widgets = append(widgets,
			MouseTarget(selectFolder(loc.file),
				Styled(styleBreadcrumbs(loc.file), Text(loc.file.name)),
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
