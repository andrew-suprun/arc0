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
	paths            []string
	scanStates       []*files.ScanState
	locations        []location
	scanResults      files.ArchiveInfos
	maps             []maps   // source, copy1, copy2, ...
	links            []*links // copy1, copy2, ...
	archiveViewLines ui.Y
}

func NewModel(paths []string) *Model {
	return &Model{
		paths:       paths,
		scanStates:  make([]*files.ScanState, len(paths)),
		scanResults: make(files.ArchiveInfos, len(paths)),
	}
}

type File struct {
	info   *files.FileInfo
	kind   fileKind
	status fileStatus
	name   string
	size   int
	files  []*File
}

type location struct {
	file       *File
	selected   *File
	lineOffset ui.Y
}

type fileKind int

const (
	regularFile fileKind = iota
	folder
)

type fileStatus int

const (
	identical fileStatus = iota
	sourceOnly
	extraCopy
	copyOnly
	discrepancy // расхождение
)

type links struct {
	sourceLinks  map[*files.FileInfo]*files.FileInfo
	reverseLinks map[*files.FileInfo]*files.FileInfo
}

type maps struct {
	byName groupByName
	byHash groupByHash
}

type groupByName map[string]*files.FileInfo
type groupByHash map[string]files.FileInfos

var (
	DefaultStyle       = ui.Style{FG: 231, BG: 17}
	styleAppTitle      = ui.Style{FG: 226, BG: 0, Bold: true, Italic: true}
	styleProgressBar   = ui.Style{FG: 231, BG: 19}
	styleArchiveHeader = ui.Style{FG: 231, BG: 8, Bold: true}
)

func styleFile(status fileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 17}
	if selected {
		result.Reverse = true
	}
	return result
}

func styleFolder(status fileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 18, Bold: true, Italic: true}
	if selected {
		result.Reverse = true
	}
	return result
}

func statusColor(status fileStatus) int {
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

func (s fileStatus) String() string {
	switch s {
	case identical:
		return "identical"
	case sourceOnly:
		return "sourceOnly"
	case copyOnly:
		return "copyOnly"
	case extraCopy:
		return "extraCopy"
	case discrepancy:
		return "discrepancy"
	}
	return "UNDEFINED"
}

func (s fileStatus) Merge(other fileStatus) fileStatus {
	if s > other {
		return s
	}
	return other
}

func (m *Model) View(event any) ui.Widget {
	m.handleEvent(event)
	return ui.Styled(DefaultStyle,
		ui.Column(ui.Flex(0),
			m.title(),
			m.scanStats(),
			m.treeView(),
		))
}

func (m *Model) handleEvent(event any) {
	switch event := event.(type) {
	case *files.ScanState:
		for i := range m.paths {
			if m.paths[i] == event.Archive {
				m.scanStates[i] = event
				break
			}
		}

	case *files.ArchiveInfo:
		for i := range m.paths {
			if m.paths[i] == event.Archive {
				m.scanStates[i] = nil
				m.scanResults[i] = event
				break
			}
		}
		doneScanning := true
		for i := range m.paths {
			if m.scanResults[i] == nil {
				doneScanning = false
				break
			}
		}
		if doneScanning {
			m.analizeArchives()
		}

	case ui.KeyEvent:
		if location := m.currentLocation(); location != nil {
			m.handleArchiveKeyEvent(event, location)
		}

	case ui.MouseEvent:
		log.Printf("EventMouse: [%d:%d]", event.Col, event.Line)

	case ui.ResizeEvent:
		// handled in App

	default:
		log.Printf("### unhandled event %#v", event)
	}
}

func (m *Model) analizeArchives() {
	m.scanStates = nil
	m.maps = make([]maps, len(m.scanResults))
	for i, scan := range m.scanResults {
		m.maps[i] = maps{
			byName: byName(scan.Files),
			byHash: byHash(scan.Files),
		}
	}

	m.links = make([]*links, len(m.scanResults)-1)
	for i, copy := range m.scanResults[1:] {
		m.links[i] = m.linkArchives(copy.Files)
	}
	m.buildFileTree()
}

func byName(infos files.FileInfos) groupByName {
	result := groupByName{}
	for _, info := range infos {
		result[info.Name] = info
	}
	return result
}

func byHash(archive files.FileInfos) groupByHash {
	result := groupByHash{}
	for _, info := range archive {
		result[info.Hash] = append(result[info.Hash], info)
	}
	return result
}

func (m *Model) buildFileTree() {

	m.locations = []location{{
		file: &File{kind: folder},
	}}

	uniqueFileNames := map[string]struct{}{}
	for _, info := range m.scanResults[0].Files {
		uniqueFileNames[info.Name] = struct{}{}
	}
	for i, copyScan := range m.scanResults[1:] {
		reverseLinks := m.links[i].reverseLinks
		for _, info := range copyScan.Files {
			if _, ok := reverseLinks[info]; !ok {
				uniqueFileNames[info.Name] = struct{}{}
			}
		}
	}

	for fullName := range uniqueFileNames {
		path := strings.Split(fullName, "/")
		name := path[len(path)-1]
		path = path[:len(path)-1]
		infos := make([]*files.FileInfo, len(m.maps))
		for i, info := range m.maps {
			infos[i] = info.byName[fullName]
		}
		for i, info := range infos {
			current := m.locations[0].file
			fileStack := []*File{current}
			if info == nil {
				continue
			}
			if i > 0 && infos[0] != nil && infos[0].Hash == info.Hash {
				continue
			}
			if i == 0 {
				current.size += info.Size
			}
			for _, dir := range path {
				sub := subFolder(current, dir)
				if i == 0 {
					sub.size += info.Size
				}
				current = sub
				fileStack = append(fileStack, current)
			}

			status := identical
			if i == 0 {
				for _, links := range m.links {
					if links.sourceLinks[info] == nil {
						status = sourceOnly
					}
				}
			} else {
				if i > 0 && infos[0] != nil {
					status = discrepancy
				} else {
					status = copyOnly
				}
			}

			currentFile := &File{
				info:   info,
				kind:   regularFile,
				status: status,
				name:   name,
				size:   info.Size,
			}
			current.files = append(current.files, currentFile)
			for _, current = range fileStack {
				current.status = status.Merge(current.status)
			}
		}
	}
	// printArchive(m.currentLocation().File, "")
}

func subFolder(dir *File, name string) *File {
	for i := range dir.files {
		if name == dir.files[i].name && dir.files[i].kind == folder {
			return dir.files[i]
		}
	}
	subFolder := &File{kind: folder, name: name}
	dir.files = append(dir.files, subFolder)
	return subFolder
}

func (a *links) String() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "Source Map:")
	for s, c := range a.sourceLinks {
		fmt.Fprintf(b, "  %s -> %s %s\n", s.Name, c.Name, s.Hash)
	}
	fmt.Fprintln(b, "Reverse Map:")
	for s, c := range a.reverseLinks {
		fmt.Fprintf(b, "  %s -> %s %s\n", s.Name, c.Name, s.Hash)
	}
	return b.String()
}

func (m *Model) linkArchives(copyInfos files.FileInfos) *links {
	result := &links{
		sourceLinks:  map[*files.FileInfo]*files.FileInfo{},
		reverseLinks: map[*files.FileInfo]*files.FileInfo{},
	}
	for _, copy := range copyInfos {
		if sources, ok := m.maps[0].byHash[copy.Hash]; ok {
			match(sources, copy, result.sourceLinks)
		}
	}

	for source, copy := range result.sourceLinks {
		result.reverseLinks[copy] = source
	}

	return result
}

func match(sources files.FileInfos, copy *files.FileInfo, sourceMap map[*files.FileInfo]*files.FileInfo) *files.FileInfo {
	for _, source := range sources {
		if copy.Name == source.Name {
			sourceMap[source] = copy
			return nil
		}
	}

	for _, source := range sources {
		tmpCopy := sourceMap[source]
		sourceBase := filepath.Base(source.Name)
		if filepath.Base(copy.Name) == sourceBase && (tmpCopy == nil || filepath.Base(tmpCopy.Name) != sourceBase) {
			sourceMap[source] = copy
			copy = tmpCopy
			break
		}
	}

	if copy == nil {
		return nil
	}

	for _, source := range sources {
		tmpCopy := sourceMap[source]
		sourceBase := filepath.Base(source.Name)
		sourceDir := filepath.Dir(source.Name)
		if filepath.Dir(copy.Name) == sourceDir &&
			(tmpCopy == nil ||
				(filepath.Base(tmpCopy.Name) != sourceBase && filepath.Dir(tmpCopy.Name) != sourceDir)) {

			sourceMap[source] = copy
			copy = tmpCopy
			break
		}
	}

	if copy == nil {
		return nil
	}

	for _, source := range sources {
		if sourceMap[source] == nil {
			sourceMap[source] = copy
			return nil
		}
	}

	return copy
}

func printArchive(archive *File, prefix string) {
	kind := "D"
	if archive.kind == regularFile {
		kind = "F"
	}
	if archive.kind == regularFile {
		log.Printf("%s%s: %s status=%v size=%v hash=%v", prefix, kind, archive.name, archive.status, archive.size, archive.info.Hash)
	} else {
		log.Printf("%s%s: %s status=%v size=%v", prefix, kind, archive.name, archive.status, archive.size)
	}
	for _, file := range archive.files {
		printArchive(file, prefix+"│ ")
	}
}

func (m *Model) handleArchiveKeyEvent(key ui.KeyEvent, loc *location) {
	switch key.Name {
	case "Home":
		loc.selected = loc.file.files[0]

	case "End":
		loc.selected = loc.file.files[len(loc.file.files)-1]

	case "PgUp":
		loc.lineOffset -= m.archiveViewLines
		if loc.lineOffset < 0 {
			loc.lineOffset = 0
		}
		idxSelected := ui.Y(0)
		foundSelected := false
		for i := 0; i < len(loc.file.files); i++ {
			if loc.file.files[i] == loc.selected {
				idxSelected = ui.Y(i)
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
		if loc.lineOffset > ui.Y(len(loc.file.files))-m.archiveViewLines {
			loc.lineOffset = ui.Y(len(loc.file.files)) - m.archiveViewLines
		}
		idxSelected := ui.Y(0)
		foundSelected := false
		for i := 0; i < len(loc.file.files); i++ {
			if loc.file.files[i] == loc.selected {
				idxSelected = ui.Y(i)
				foundSelected = true
				break
			}
		}
		if foundSelected {
			idxSelected += m.archiveViewLines
			if idxSelected > ui.Y(len(loc.file.files))-1 {
				idxSelected = ui.Y(len(loc.file.files)) - 1
			}
			loc.selected = loc.file.files[idxSelected]
		}

	case "Up":
		if loc.selected != nil {
			for i, file := range loc.file.files {
				if file == loc.selected && i > 0 {
					loc.selected = loc.file.files[i-1]
					break
				}
			}
		} else {
			loc.selected = loc.file.files[len(loc.file.files)-1]
		}

	case "Down":
		if loc.selected != nil {
			for i, file := range loc.file.files {
				if file == loc.selected && i+1 < len(loc.file.files) {
					loc.selected = loc.file.files[i+1]
					break
				}
			}
		} else {
			loc.selected = loc.file.files[0]
		}

	}
}

func (m *Model) currentLocation() *location {
	if len(m.locations) == 0 {
		return nil
	}
	return &m.locations[len(m.locations)-1]
}

func (m *Model) title() ui.Widget {
	return ui.Row(
		ui.Styled(styleAppTitle, ui.Text(" АРХИВАТОР", 4, 1)),
	)
}

func (m *Model) scanStats() ui.Widget {
	if m.scanStates == nil {
		return ui.NullWidget{}
	}
	forms := []ui.Widget{}
	for i := range m.scanStates {
		if m.scanStates[i] != nil {
			forms = append(forms, scanStatsForm(m.scanStates[i]))
		}
	}
	forms = append(forms, ui.Spacer{})
	return ui.Column(1, forms...)
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

func (m *Model) treeView() ui.Widget {
	if len(m.locations) == 0 {
		return ui.NullWidget{}
	}

	return ui.Column(ui.Flex(1),
		ui.Styled(styleArchiveHeader,
			ui.Row(ui.Text(" Статус", 7, 0), ui.Text("  Документ", 21, 1), ui.Text(" Время Изменения", 21, 0), ui.Text("            Размер ", 19, 0)),
		),
		ui.Sized(ui.MakeConstraints(0, 1, 0, 1),
			func(width ui.X, height ui.Y) ui.Widget {
				m.archiveViewLines = height
				location := m.currentLocation()
				if location.lineOffset > ui.Y(len(location.file.files)+1-int(height)) {
					location.lineOffset = ui.Y(len(location.file.files) + 1 - int(height))
				}
				if location.lineOffset < 0 {
					location.lineOffset = 0
				}
				if location.selected != nil {
					idx := ui.Y(-1)
					for i := range location.file.files {
						if location.selected == location.file.files[i] {
							idx = ui.Y(i)
							break
						}
					}
					if idx >= 0 {
						if location.lineOffset > idx {
							location.lineOffset = idx
						}
						if location.lineOffset < idx+1-height {
							location.lineOffset = idx + 1 - height
						}
					}
				}
				rows := make([]ui.Widget, height)
				i := 0
				var file *File
				for i, file = range location.file.files[location.lineOffset:] {
					if i >= len(rows) {
						break
					}
					if file.kind == regularFile {
						rows[i] = ui.Styled(styleFile(file.status, location.selected == file),
							ui.Row(
								ui.Text(file.status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text(file.info.ModTime.Format(time.DateTime), 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.size), 18, 0),
							),
						)
					} else {
						rows[i] = ui.Styled(styleFolder(file.status, location.selected == file),
							ui.Row(
								ui.Text(file.status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text("<Каталог>", 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.size), 18, 0),
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
