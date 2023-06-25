package controller

import (
	"arch/model"
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

func (c *controller) view() w.Widget {
	return w.Column(col,
		c.title(),
		c.folderView(),
		c.progress(),
	)
}

func (c *controller) title() w.Widget {
	return w.Row(row,
		w.Styled(styleAppTitle, w.Text(" Archiver").Flex(1)),
	)
}

func (c *controller) folderView() w.Widget {
	folder := c.folders[c.currentPath]
	return w.Column(col,
		c.breadcrumbs(),
		w.Styled(styleArchiveHeader,
			w.Row(row,
				w.MouseTarget(sortByStatus, w.Text(" St"+c.sortIndicator(sortByStatus)).Width(3+len(c.archives))),
				w.MouseTarget(sortByName, w.Text(" Document"+c.sortIndicator(sortByName)).Width(20).Flex(1)),
				w.MouseTarget(sortByTime, w.Text("  Date Modified"+c.sortIndicator(sortByTime)).Width(19)),
				w.MouseTarget(sortBySize, w.Text(fmt.Sprintf("%22s", "Size"+c.sortIndicator(sortBySize)+" "))),
			),
		),
		w.Scroll(model.Scroll{}, w.Constraint{Size: w.Size{Width: 0, Height: 0}, Flex: w.Flex{X: 1, Y: 1}},
			func(size w.Size) w.Widget {
				c.fileTreeLines = size.Height
				if folder.lineOffset > len(folder.entries)+1-size.Height {
					folder.lineOffset = len(folder.entries) + 1 - size.Height
				}
				if folder.lineOffset < 0 {
					folder.lineOffset = 0
				}
				rows := []w.Widget{}
				i := 0
				var file *model.File
				for i, file = range folder.entries[folder.lineOffset:] {
					if i >= size.Height {
						break
					}
					rows = append(rows, w.Styled(styleFile(file, c.folders[c.currentPath].selected == file),
						w.MouseTarget(selectFile(file), w.Row(row,
							c.fileStatus(file)...,
						)),
					))
				}
				rows = append(rows, w.Spacer{})
				return w.Column(col, rows...)
			},
		),
	)
}

func (c *controller) fileStatus(file *model.File) []w.Widget {
	result := []w.Widget{w.Text(" " + file.Status.String()).Width(11)}

	if file.Kind == model.FileRegular {
		result = append(result, w.Text("   "))
	} else {
		result = append(result, w.Text(" ▶ "))
	}
	result = append(result, w.Text(name(file.Name)).Width(20).Flex(1))
	result = append(result, w.Text("  "))
	result = append(result, w.Text(file.ModTime.Format(time.DateTime)))
	result = append(result, w.Text("  "))
	result = append(result, w.Text(formatSize(file.Size)).Width(18))
	return result
}

func (c *controller) sortIndicator(column sortColumn) string {
	folder := c.folders[c.currentPath]
	if column == folder.sortColumn {
		if folder.sortAscending[column] {
			return " ▲"
		}
		return " ▼"
	}
	return ""
}

func (c *controller) breadcrumbs() w.Widget {
	names := strings.Split(c.currentPath, "/")
	widgets := make([]w.Widget, 0, len(names)*2+2)
	widgets = append(widgets, w.MouseTarget(selectFolder(c.folders[""].info),
		w.Styled(styleBreadcrumbs, w.Text(" Root")),
	))
	for i := range names {
		widgets = append(widgets, w.Text(" / "))
		widgets = append(widgets,
			w.MouseTarget(selectFolder(c.folders[filepath.Join(names[:i+1]...)].info),
				w.Styled(styleBreadcrumbs, w.Text(names[i])),
			),
		)
	}
	widgets = append(widgets, w.Spacer{})
	return w.Row(row, widgets...)
}

type progressInfo struct {
	progressLabel string
	labelWidth    int
	value         float64
}

func (c *controller) progress() w.Widget {
	rootLen := 0
	for path := range c.archives {
		if rootLen < len(path) {
			rootLen = len(path)
		}
	}
	progressInfos := make([]progressInfo, 0, len(c.archives)+1)
	for _, root := range c.roots {
		archive := c.archives[root]
		if archive.progress.ProgressState == model.HashingFileTree {
			progressInfos = append(progressInfos, progressInfo{
				progressLabel: " Hashing: " + root,
				labelWidth:    11 + rootLen,
				value:         float64(archive.progress.TotalHashed) / float64(archive.totalSize),
			})
		}
	}
	if c.fileHandler != nil && c.copySize > 0 {
		progressInfos = append(progressInfos, progressInfo{
			progressLabel: " Copying: ",
			labelWidth:    11,
			value:         float64(c.totalCopied+c.fileCopied) / float64(c.copySize),
		})
	}
	stats := []w.Widget{}
	for _, progress := range progressInfos {
		stats = append(stats,
			w.Row(w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}},
				w.Text(progress.progressLabel).Width(progress.labelWidth),
				w.Text(fmt.Sprintf(" %6.2f%%", progress.value*100)), w.Text(" "),
				w.Styled(styleProgressBar,
					w.ProgressBar(progress.value),
				),
				w.Text(" "),
			),
		)
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

func styleFile(file *model.File, selected bool) w.Style {
	bg, flags := byte(17), w.Flags(0)
	if file.Kind == model.FileFolder {
		bg = byte(18)
	}
	result := w.Style{FG: statusColor(file.Status), BG: bg, Flags: flags}
	if selected {
		result.Flags |= w.Reverse
	}
	return result
}

var styleBreadcrumbs = w.Style{FG: 250, BG: 17, Flags: w.Bold + w.Italic}

func statusColor(status model.Status) byte {
	switch status {
	case model.Identical:
		return 250
	case model.Pending:
		return 214
	case model.Resolved:
		return 82
	case model.Duplicate:
		return 196
	case model.Absent:
		return 196
	}
	return 231
}
