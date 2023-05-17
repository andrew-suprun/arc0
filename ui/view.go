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

type model struct {
	events           chan any
	paths            []string
	scanStates       []*files.ScanState
	locations        []location
	scanResults      files.ArchiveInfos
	maps             []maps   // source, copy1, copy2, ...
	links            []*links // copy1, copy2, ...
	screenSize       Size
	archiveViewLines int
	ctx              *Context
	lastMouseEvent   device.MouseEvent
}

func Run(r device.Device, events chan any, paths []string) {
	m := &model{
		events:      events,
		paths:       paths,
		scanStates:  make([]*files.ScanState, len(paths)),
		scanResults: make(files.ArchiveInfos, len(paths)),
		ctx:         &Context{Device: r, Style: defaultStyle},
	}

	for m.handleEvent(<-events) {
		m.ctx.Reset()
		Column(0,
			m.title(),
			m.scanStats(),
			m.treeView(),
			m.statusLine(),
		).Render(m.ctx, Position{0, 0}, m.screenSize)
		m.ctx.Device.Render()
	}
}

type File struct {
	info   *files.FileInfo
	kind   fileKind
	status fileStatus
	path   string
	name   string
	size   int
	files  []*File
}

type location struct {
	file       *File
	selected   *File
	lineOffset int
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
	defaultStyle       = device.Style{FG: 231, BG: 17}
	styleAppTitle      = device.Style{FG: 226, BG: 0, Flags: device.Bold + device.Italic}
	styleStatusLine    = device.Style{FG: 226, BG: 0}
	styleProgressBar   = device.Style{FG: 231, BG: 19}
	styleArchiveHeader = device.Style{FG: 231, BG: 8, Flags: device.Bold}
)

type selectFile *File

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

func (m *model) handleEvent(event any) bool {
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

	case device.ResizeEvent:
		m.screenSize = Size(event)

	case device.KeyEvent:
		if event.Name == "Ctrl+C" {
			return false
		}
		m.handleArchiveKeyEvent(event)

	case device.MouseEvent:
		for _, target := range m.ctx.MouseTargetAreas {
			if target.Pos.X <= event.X && target.Pos.X+target.Size.Width > event.X &&
				target.Pos.Y <= event.Y && target.Pos.Y+target.Size.Height > event.Y {

				if file, ok := target.Command.(selectFile); ok {
					m.currentLocation().selected = file
				}
				last := m.lastMouseEvent
				if last.X == event.X && last.Y == event.Y &&
					last.Button == event.Button &&
					last.ButtonModifier == event.ButtonModifier {

					if event.Time.Sub(last.Time).Seconds() < 0.5 {
						m.enter()
					}
				}
				m.lastMouseEvent = event
			}
		}
	case device.ScrollEvent:
		if event.Direction == device.ScrollUp {
			// m.currentLocation().lineOffset++
			m.up()
		} else {
			// m.currentLocation().lineOffset--
			m.down()
		}

	default:
		log.Printf("### unhandled event %#v", event)
	}
	return true
}

func (m *model) analizeArchives() {
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

func (m *model) buildFileTree() {
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
			for pathIdx, dir := range path {
				sub := subFolder(current, dir)
				if i == 0 {
					sub.size += info.Size
				}
				if sub.path == "" {
					sub.path = info.Archive + "/" + filepath.Join(path[:pathIdx+1]...)
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
				path:   filepath.Join(info.Archive, info.Name),
				name:   name,
				size:   info.Size,
			}
			current.files = append(current.files, currentFile)
			for _, current = range fileStack {
				current.status = status.Merge(current.status)
			}
		}
	}
	PrintArchive(m.currentLocation().file, "")
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

func (m *model) linkArchives(copyInfos files.FileInfos) *links {
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

func PrintArchive(archive *File, prefix string) {
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
		PrintArchive(file, prefix+"│ ")
	}
}

func (m *model) handleArchiveKeyEvent(key device.KeyEvent) {
	loc := m.currentLocation()
	if loc == nil {
		return
	}

	switch key.Name {
	case "Enter":
		m.enter()

	case "Rune[R]", "Rune[r]":
		exec.Command("open", "-R", loc.selected.path).Start()

	case "Esc":
		if len(m.locations) > 1 {
			m.locations = m.locations[:len(m.locations)-1]
		}

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
}

func (m *model) enter() {
	loc := m.currentLocation()
	if loc.selected != nil && loc.selected.kind == folder {
		m.locations = append(m.locations, location{file: loc.selected})
	} else {
		fileName := filepath.Join(loc.selected.info.Archive, loc.selected.info.Name)
		exec.Command("open", fileName).Start()
	}
}

func (m *model) up() {
	loc := m.currentLocation()
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
}

func (m *model) down() {
	loc := m.currentLocation()
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

func (m *model) currentLocation() *location {
	if len(m.locations) == 0 {
		return nil
	}
	return &m.locations[len(m.locations)-1]
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
	log.Println(Text(filepath.Base(state.Name)).Flex(1))
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
		Styled(styleArchiveHeader,
			Row(Text(" Статус").Width(7), Text("  Документ").Width(21).Flex(1), Text(" Время Изменения").Width(21), Text("            Размер ").Width(19)),
		),
		Scroll(nil, Constraint{Size{0, 0}, Flex{1, 1}},
			func(size Size) Widget {
				m.archiveViewLines = size.Height
				location := m.currentLocation()
				if location.lineOffset > len(location.file.files)+1-size.Height {
					location.lineOffset = len(location.file.files) + 1 - size.Height
				}
				if location.lineOffset < 0 {
					location.lineOffset = 0
				}
				if location.selected != nil {
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
						if location.lineOffset < idx+1-size.Height {
							location.lineOffset = idx + 1 - size.Height
						}
					}
				}
				rows := make([]Widget, size.Height)
				i := 0
				var file *File
				for i, file = range location.file.files[location.lineOffset:] {
					if i >= len(rows) {
						break
					}
					rows[i] = Styled(styleFile(file, location.selected == file),
						MouseTarget(selectFile(file), Row(
							Text(" "+file.status.String()).Width(7),
							Text("  "),
							Text(file.name).Width(20).Flex(1),
							Text("  "),
							Text(modTime(file)).Width(19),
							Text("  "),
							Text(formatSize(file.size)).Width(18),
						)),
					)
				}
				for i++; i < size.Height; i++ {
					rows[i] = Text("").Flex(1)
				}
				return Column(0, rows...)
			},
		),
	)
}

func styleFile(file *File, selected bool) device.Style {
	bg, flags := byte(17), device.Flags(0)
	if file.kind == folder {
		bg, flags = byte(18), device.Bold+device.Italic
	}
	result := device.Style{FG: statusColor(file.status), BG: bg, Flags: flags}
	if selected {
		result.Flags |= device.Reverse
	}
	return result
}

func modTime(file *File) string {
	if file.kind == regularFile {
		return file.info.ModTime.Format(time.DateTime)
	}
	return "<Каталог>"
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
