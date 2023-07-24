package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

var (
	styleDefault       = w.Style{FG: 226, BG: 17}
	styleAppTitle      = w.Style{FG: 226, BG: 0, Flags: w.Bold + w.Italic}
	styleStatusLine    = w.Style{FG: 230, BG: 0, Flags: w.Italic}
	styleArchive       = w.Style{FG: 226, BG: 0, Flags: w.Bold}
	styleProgressBar   = w.Style{FG: 231, BG: 19}
	styleArchiveHeader = w.Style{FG: 231, BG: 8, Flags: w.Bold}
)

var (
	rowConstraint = w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}}
	colConstraint = w.Constraint{Size: w.Size{Width: 0, Height: 0}, Flex: w.Flex{X: 1, Y: 1}}
)

func (c *controller) RootWidget() w.Widget {
	return w.Styled(styleDefault,
		w.Column(colConstraint,
			c.title(),
			c.folderWidget(),
			c.progress(),
			c.fileStats(),
		),
	)
}

func (c *controller) title() w.Widget {
	return w.Row(rowConstraint,
		w.Styled(styleAppTitle, w.Text(" Archiver").Flex(1)),
	)
}

func (s *controller) folderWidget() w.Widget {
	return w.Column(colConstraint,
		s.breadcrumbs(),
		w.Styled(styleArchiveHeader,
			w.Row(rowConstraint,
				w.Text(" Status").Width(13),
				w.MouseTarget(m.SortByName, w.Text(" Document"+s.sortIndicator(m.SortByName)).Width(20).Flex(1)),
				w.MouseTarget(m.SortByTime, w.Text("  Date Modified"+s.sortIndicator(m.SortByTime)).Width(19)),
				w.MouseTarget(m.SortBySize, w.Text(fmt.Sprintf("%22s", "Size"+s.sortIndicator(m.SortBySize)+" "))),
			),
		),
		w.Scroll(m.Scroll{}, w.Constraint{Size: w.Size{Width: 0, Height: 0}, Flex: w.Flex{X: 1, Y: 1}},
			func(size w.Size) w.Widget {
				folder := s.currentFolder()
				sorted := folder.sort()
				log.Printf("folderWidget: folder: %v", folder)
				s.fileTreeLines = size.Height
				if folder.offsetIdx > len(folder.entries)+1-size.Height {
					folder.offsetIdx = len(folder.entries) + 1 - size.Height
				}
				if folder.offsetIdx < 0 {
					folder.offsetIdx = 0
				}
				rows := []w.Widget{}
				for i, file := range sorted[folder.offsetIdx:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, w.Styled(s.styleFile(file, folder.selectedEntry == file),
						w.MouseTarget(m.SelectFile(file.Id), w.Row(rowConstraint,
							s.fileRow(file)...,
						)),
					))
				}
				rows = append(rows, w.Spacer{})
				return w.Column(colConstraint, rows...)
			},
		),
	)
}

func (s *controller) fileRow(file *m.File) []w.Widget {
	result := []w.Widget{w.Text(stateString(file.State)).Width(11)}

	if file.Kind == m.FileRegular {
		result = append(result, w.Text("   "))
	} else {
		result = append(result, w.Text(" ▶ "))
	}
	result = append(result, w.Text(file.Base.String()).Width(20).Flex(1))
	result = append(result, w.Text("  "))
	result = append(result, w.Text(file.ModTime.Format(time.DateTime)))
	result = append(result, w.Text("  "))
	result = append(result, w.Text(formatSize(file.Size)).Width(18))
	return result
}

func stateString(state m.State) string {
	switch state {
	case m.Initial:
		return ""
	case m.Resolved:
		return ""
	case m.Pending:
		return " Pending"
	case m.Duplicate:
		return " Duplicate"
	case m.Absent:
		return " Absent"
	}
	return "UNKNOWN"
}

func (c *controller) sortIndicator(column m.SortColumn) string {
	folder := c.currentFolder()
	if column == folder.sortColumn {
		if folder.sortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func (c *controller) breadcrumbs() w.Widget {
	names := strings.Split(c.currentPath.String(), "/")
	widgets := make([]w.Widget, 0, len(names)*2+2)
	widgets = append(widgets, w.MouseTarget(m.SelectFolder(""),
		w.Styled(styleBreadcrumbs, w.Text(" Root")),
	))
	for i := range names {
		widgets = append(widgets, w.Text(" / "))
		widgets = append(widgets,
			w.MouseTarget(m.SelectFolder(m.Path(filepath.Join(names[:i+1]...))),
				w.Styled(styleBreadcrumbs, w.Text(names[i])),
			),
		)
	}
	widgets = append(widgets, w.Spacer{})
	return w.Row(rowConstraint, widgets...)
}

func (c *controller) progressXXX() []w.ProgressInfo {
	infos := []w.ProgressInfo{}
	archive := c.archives[c.origin]
	if archive.progressState == m.ProgressScanned && c.copySize > 0 {
		infos = append(infos, w.ProgressInfo{
			Root:          c.origin,
			Tab:           " Copying",
			Value:         float64(c.totalCopiedSize+uint64(c.fileCopiedSize)) / float64(c.copySize),
			Speed:         c.copySpeed,
			TimeRemaining: c.timeRemaining,
		})
	}
	var tab string
	var value float64
	for _, root := range c.roots {
		archive := c.archives[root]
		if archive.progressState == m.ProgressScanned {
			continue
		}
		tab = " Hashing"
		value = float64(archive.totalHashed+archive.fileHashed) / float64(archive.totalSize)
		infos = append(infos, w.ProgressInfo{
			Root:          root,
			Tab:           tab,
			Value:         value,
			Speed:         archive.speed,
			TimeRemaining: archive.timeRemaining,
		})
	}
	return infos
}

func (s *controller) progress() w.Widget {
	tabWidth := 0
	rootWidth := 0
	for _, progress := range s.progressInfos {
		if tabWidth < len(progress.Tab) {
			tabWidth = len(progress.Tab)
		}
		if rootWidth < len(progress.Root) {
			rootWidth = len(progress.Root)
		}
	}

	stats := []w.Widget{}
	for _, progress := range s.progressInfos {
		stats = append(stats,
			w.Row(w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}},
				w.Text(progress.Tab).Width(tabWidth),
				w.Text(" "),
				w.Styled(styleArchive, w.Text(progress.Root.String()).Width(rootWidth)),
				w.Text(fmt.Sprintf(" %6.2f%%", progress.Value*100)),
				w.Text(fmt.Sprintf(" %5.1f Mb/S", progress.Speed)),
				w.Text(fmt.Sprintf(" ETA %6s", progress.TimeRemaining.Truncate(time.Second))), w.Text(" "),
				w.Styled(styleProgressBar, w.ProgressBar(progress.Value)),
				w.Text(" "),
			),
		)
	}
	return w.Styled(styleStatusLine,
		w.Column(w.Constraint{Size: w.Size{Width: 0, Height: len(stats)}, Flex: w.Flex{X: 1, Y: 0}}, stats...),
	)
}
