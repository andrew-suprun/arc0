package model

import (
	"arch/events"
	w "arch/widgets"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	styleAppTitle      = w.Style{FG: 226, BG: 0, Flags: w.Bold + w.Italic}
	styleStatusLine    = w.Style{FG: 226, BG: 0}
	styleProgressBar   = w.Style{FG: 231, BG: 19}
	styleArchiveHeader = w.Style{FG: 231, BG: 8, Flags: w.Bold}
)

var (
	row = w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}}
	col = w.Constraint{Size: w.Size{Width: 0, Height: 0}, Flex: w.Flex{X: 1, Y: 1}}
)

func (m *model) view() w.Widget {
	return w.Column(col,
		m.title(),
		m.folderView(),
		m.scanProgress(),
	)
}

func (m *model) title() w.Widget {
	return w.Row(row,
		w.Styled(styleAppTitle, w.Text(" Archiver").Flex(1)),
	)
}

func (m *model) folderView() w.Widget {
	folder := m.folders[m.currentPath]
	return w.Column(col,
		m.breadcrumbs(),
		w.Styled(styleArchiveHeader,
			w.Row(row,
				w.MouseTarget(sortByStatus, w.Text(" State"+sortIndicator(m, sortByStatus)).Width(13)),
				w.MouseTarget(sortByName, w.Text("    Document"+sortIndicator(m, sortByName)).Width(20).Flex(1)),
				w.MouseTarget(sortByTime, w.Text("  Date Modified"+sortIndicator(m, sortByTime)).Width(19)),
				w.MouseTarget(sortBySize, w.Text(fmt.Sprintf("%22s", "Size"+sortIndicator(m, sortBySize)+" "))),
			),
		),
		w.Scroll(events.Scroll{}, w.Constraint{Size: w.Size{Width: 0, Height: 0}, Flex: w.Flex{X: 1, Y: 1}},
			func(size w.Size) w.Widget {
				m.fileTreeLines = size.Height
				if folder.lineOffset > len(folder.entries)+1-size.Height {
					folder.lineOffset = len(folder.entries) + 1 - size.Height
				}
				if folder.lineOffset < 0 {
					folder.lineOffset = 0
				}
				rows := []w.Widget{}
				i := 0
				var file *File
				for i, file = range folder.entries[folder.lineOffset:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, w.Styled(styleFile(file, m.folders[m.currentPath].selected == file),
						w.MouseTarget(selectFile(file), w.Row(row,
							w.Text(" "+repr(file.Status)).Width(13),
							w.Text("  "),
							w.Text(displayName(file)).Width(20).Flex(1),
							w.Text("  "),
							w.Text(file.ModTime.Format(time.DateTime)),
							w.Text("  "),
							w.Text(formatSize(file.Size)).Width(18),
						)),
					))
				}
				rows = append(rows, w.Spacer{})
				return w.Column(col, rows...)
			},
		),
	)
}

func displayName(file *File) string {
	if file.Kind == FileFolder {
		return "▶ " + file.Name
	}
	return "  " + file.Name
}

func sortIndicator(m *model, column sortColumn) string {
	folder := m.folders[m.currentPath]
	if column == folder.sortColumn {
		if folder.sortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func (m *model) breadcrumbs() w.Widget {
	names := strings.Split(m.currentPath, "/")
	widgets := make([]w.Widget, 0, len(names)*2+2)
	widgets = append(widgets, w.MouseTarget(selectFolder(m.folders[""].info),
		w.Styled(styleBreadcrumbs, w.Text(" Root")),
	))
	for i := range names {
		widgets = append(widgets, w.Text(" / "))
		widgets = append(widgets,
			w.MouseTarget(selectFolder(m.folders[filepath.Join(names[:i+1]...)].info),
				w.Styled(styleBreadcrumbs, w.Text(names[i])),
			),
		)
	}
	widgets = append(widgets, w.Spacer{})
	return w.Row(row, widgets...)
}

func (m *model) scanProgress() w.Widget {
	pathLen := 0
	for _, archive := range m.archives {
		if pathLen < len(archive.archivePath) {
			pathLen = len(archive.archivePath)
		}
	}
	stats := []w.Widget{}
	for i, archive := range m.archives {
		state := archive.scanState
		if state.ScanState == events.HashFileTree {
			stats = append(stats,
				w.Row(w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}},
					w.Text(" Scanning: "+m.archives[i].archivePath).Width(pathLen+11),
					w.Text(fmt.Sprintf(" %5.1f%%", state.ScanProgress*100)), w.Text(" "),
					w.Styled(styleProgressBar,
						w.ProgressBar(state.ScanProgress),
					),
					w.Text(" "),
				),
			)
		}
	}
	return w.Styled(styleStatusLine,
		w.Column(w.Constraint{Size: w.Size{Width: 0, Height: len(stats)}, Flex: w.Flex{X: 1, Y: 0}}, stats...),
	)
}

func formatSize(size uint64) string {
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

func styleFile(file *File, selected bool) w.Style {
	bg, flags := byte(17), w.Flags(0)
	if file.Kind == FileFolder {
		bg = byte(18)
	}
	result := w.Style{FG: statusColor(file.Status), BG: bg, Flags: flags}
	if selected {
		result.Flags |= w.Reverse
	}
	return result
}

var styleBreadcrumbs = w.Style{FG: 250, BG: 17, Flags: w.Bold + w.Italic}

func statusColor(status FileStatus) byte {
	switch status {
	case Identical:
		return 250
	case SourceOnly:
		return 82
	case CopyOnly:
		return 196
	}
	return 231
}

func repr(status FileStatus) string {
	switch status {
	case Identical:
		return ""
	case SourceOnly:
		return "Оригинал"
	case CopyOnly:
		return "Только Копия"
	}
	return "UNDEFINED"
}