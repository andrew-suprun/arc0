package controller

import (
	m "arch/model"
	w "arch/widgets"
	"fmt"
	"strings"
)

func (c *controller) fileStats() w.Widget {
	if c.duplicateFiles == 0 && c.absentFiles == 0 && c.pendingFiles == 0 {
		return w.Text(" All Clear").Flex(1)
	}
	stats := []w.Widget{w.Text(" Stats:")}
	if c.duplicateFiles > 0 {
		stats = append(stats, w.Text(fmt.Sprintf(" Duplicates: %d", c.duplicateFiles)))
	}
	if c.absentFiles > 0 {
		stats = append(stats, w.Text(fmt.Sprintf(" Absent: %d", c.absentFiles)))
	}
	if c.pendingFiles > 0 {
		stats = append(stats, w.Text(fmt.Sprintf(" Pending: %d", c.pendingFiles)))
	}
	stats = append(stats, w.Text("").Flex(1))
	stats = append(stats, w.Text(fmt.Sprintf(" FPS: %d ", c.fps)))
	return w.Styled(
		styleAppTitle,
		w.Row(w.Constraint{Size: w.Size{Width: 0, Height: 1}, Flex: w.Flex{X: 1, Y: 0}}, stats...),
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

func (c *controller) styleFile(file *m.File, selected bool) w.Style {
	bg, flags := byte(17), w.Flags(0)
	if file.Kind == m.FileFolder {
		bg = byte(18)
	}
	result := w.Style{FG: c.statusColor(file), BG: bg, Flags: flags}
	if selected {
		result.Flags |= w.Reverse
	}
	return result
}

var styleBreadcrumbs = w.Style{FG: 250, BG: 17, Flags: w.Bold + w.Italic}

func (c *controller) statusColor(file *m.File) byte {
	switch file.State {
	case m.Initial:
		return 248
	case m.Resolved:
		return 195
	case m.Pending:
		return 214
	case m.Duplicate, m.Absent:
		return 196
	}
	return 231
}
