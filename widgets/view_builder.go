package widgets

import (
	m "arch/model"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

var (
	styleDefault       = Style{FG: 226, BG: 18}
	styleAppTitle      = Style{FG: 226, BG: 0, Flags: Bold + Italic}
	styleStatusLine    = Style{FG: 230, BG: 0, Flags: Italic}
	styleArchive       = Style{FG: 226, BG: 0, Flags: Bold}
	styleProgressBar   = Style{FG: 231, BG: 19}
	styleArchiveHeader = Style{FG: 231, BG: 8, Flags: Bold}
)

var (
	rowConstraint = Constraint{Size: Size{Width: 0, Height: 1}, Flex: Flex{X: 1, Y: 0}}
	colConstraint = Constraint{Size: Size{Width: 0, Height: 0}, Flex: Flex{X: 1, Y: 1}}
)

func (s *View) Render() Widget {
	return Styled(styleDefault,
		Column(colConstraint,
			s.title(),
			s.folderView(),
			s.progress(),
			s.fileStats(),
		),
	)
}

func (c *View) title() Widget {
	return Row(rowConstraint,
		Styled(styleAppTitle, Text(" Archiver").Flex(1)),
	)
}

func (s *View) folderView() Widget {
	return Column(colConstraint,
		s.breadcrumbs(),
		Styled(styleArchiveHeader,
			Row(rowConstraint,
				Text(" Status").Width(13),
				MouseTarget(SortByName, Text(" Document"+s.sortIndicator(SortByName)).Width(20).Flex(1)),
				MouseTarget(SortByTime, Text("  Date Modified"+s.sortIndicator(SortByTime)).Width(19)),
				MouseTarget(SortBySize, Text(fmt.Sprintf("%22s", "Size"+s.sortIndicator(SortBySize)+" "))),
			),
		),
		Scroll(m.Scroll{}, Constraint{Size: Size{Width: 0, Height: 0}, Flex: Flex{X: 1, Y: 1}},
			func(size Size) Widget {
				s.FileTreeLines = size.Height
				if s.OffsetIdx > len(s.Entries)+1-size.Height {
					s.OffsetIdx = len(s.Entries) + 1 - size.Height
				}
				if s.OffsetIdx < 0 {
					s.OffsetIdx = 0
				}
				rows := []Widget{}
				for i, file := range s.Entries[s.OffsetIdx:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, Styled(s.styleFile(file, s.SelectedId == file.Id),
						MouseTarget(m.SelectFile(file.Id), Row(rowConstraint,
							s.fileRow(file)...,
						)),
					))
				}
				rows = append(rows, Spacer{})
				return Column(colConstraint, rows...)
			},
		),
	)
}

func (s *View) fileRow(file *File) []Widget {
	result := []Widget{Text(statusString(file)).Width(11)}

	if file.Kind == FileRegular {
		result = append(result, Text("   "))
	} else {
		result = append(result, Text(" ▶ "))
	}
	result = append(result, Text(file.Base.String()).Width(20).Flex(1))
	result = append(result, Text("  "))
	result = append(result, Text(file.ModTime.Format(time.DateTime)))
	result = append(result, Text("  "))
	result = append(result, Text(formatSize(file.Size)).Width(18))
	return result
}

func statusString(file *File) string {
	switch file.State {
	case Resolved:
		return ""
	case Pending:
		return " Pending"
	case Duplicate:
		return " Duplicate"
	case Absent:
		return " Absent"
	}
	return "UNKNOWN"
}

func (s *View) sortIndicator(column SortColumn) string {
	if column == s.SortColumn {
		if s.SortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func (c *View) breadcrumbs() Widget {
	names := strings.Split(c.CurrentPath.String(), "/")
	widgets := make([]Widget, 0, len(names)*2+2)
	widgets = append(widgets, MouseTarget(m.SelectFolder(""),
		Styled(styleBreadcrumbs, Text(" Root")),
	))
	for i := range names {
		widgets = append(widgets, Text(" / "))
		widgets = append(widgets,
			MouseTarget(m.SelectFolder(m.Path(filepath.Join(names[:i+1]...))),
				Styled(styleBreadcrumbs, Text(names[i])),
			),
		)
	}
	widgets = append(widgets, Spacer{})
	return Row(rowConstraint, widgets...)
}

func (s *View) progress() Widget {
	tabWidth := 0
	rootWidth := 0
	for _, progress := range s.Progress {
		if tabWidth < len(progress.Tab) {
			tabWidth = len(progress.Tab)
		}
		if rootWidth < len(progress.Root) {
			rootWidth = len(progress.Root)
		}
	}

	stats := []Widget{}
	for _, progress := range s.Progress {
		stats = append(stats,
			Row(Constraint{Size: Size{Width: 0, Height: 1}, Flex: Flex{X: 1, Y: 0}},
				Text(progress.Tab).Width(tabWidth),
				Text(" "),
				Styled(styleArchive, Text(progress.Root.String()).Width(rootWidth)),
				Text(fmt.Sprintf(" %6.2f%%", progress.Value*100)), Text(" "),
				Styled(styleProgressBar, ProgressBar(progress.Value)),
				Text(" "),
			),
		)
	}
	return Styled(styleStatusLine,
		Column(Constraint{Size: Size{Width: 0, Height: len(stats)}, Flex: Flex{X: 1, Y: 0}}, stats...),
	)
}

func (s *View) fileStats() Widget {
	if s.DuplicateFiles == 0 && s.AbsentFiles == 0 && s.PendingFiles == 0 {
		return Text(" All Clear").Flex(1)
	}
	stats := []Widget{Text(" Stats:")}
	if s.DuplicateFiles > 0 {
		stats = append(stats, Text(fmt.Sprintf(" Duplicates: %d", s.DuplicateFiles)))
	}
	if s.AbsentFiles > 0 {
		stats = append(stats, Text(fmt.Sprintf(" Absent: %d", s.AbsentFiles)))
	}
	if s.PendingFiles > 0 {
		stats = append(stats, Text(fmt.Sprintf(" Pending: %d", s.PendingFiles)))
	}
	stats = append(stats, Text("").Flex(1))
	stats = append(stats, Text(fmt.Sprintf(" FPS: %d ", s.FPS)))
	return Styled(
		styleAppTitle,
		Row(Constraint{Size: Size{Width: 0, Height: 1}, Flex: Flex{X: 1, Y: 0}}, stats...),
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

func (c *View) styleFile(file *File, selected bool) Style {
	bg, flags := byte(17), Flags(0)
	if file.Kind == FileFolder {
		bg = byte(18)
	}
	result := Style{FG: c.statusColor(file), BG: bg, Flags: flags}
	if selected {
		result.Flags |= Reverse
	}
	return result
}

var styleBreadcrumbs = Style{FG: 250, BG: 17, Flags: Bold + Italic}

func (c *View) statusColor(file *File) byte {
	switch file.State {
	case Resolved:
		return 195
	case Pending:
		return 214
	case Duplicate, Absent:
		return 196
	}
	return 231
}
