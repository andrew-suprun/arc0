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
	log.Println("NewModel: paths =", paths)
	return &Model{
		paths:       paths,
		scanStates:  make([]*files.ScanState, len(paths)),
		scanResults: make(files.ArchiveInfos, len(paths)),
	}
}

type File struct {
	Info   *files.FileInfo
	Kind   FileKind
	Status FileStatus
	Name   string
	Size   int
	Files  []*File
}

type location struct {
	File       *File
	Selected   *File
	LineOffset ui.Y
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

func styleFile(status FileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 17}
	if selected {
		result.Reverse = true
	}
	return result
}

func styleFolder(status FileStatus, selected bool) ui.Style {
	result := ui.Style{FG: statusColor(status), BG: 18, Bold: true, Italic: true}
	if selected {
		result.Reverse = true
	}
	return result
}

func statusColor(status FileStatus) int {
	switch status {
	case Identical:
		return 250
	case SourceOnly:
		return 82
	case ExtraCopy:
		return 226
	case CopyOnly:
		return 214
	case Discrepancy:
		return 196
	}
	return 231
}

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

func (m *Model) View(event any) ui.Widget {
	m.handleEvent(event)
	log.Printf("### View %T: locations %d", event, len(m.locations))
	return ui.Styled(DefaultStyle,
		ui.Column(ui.Flex(0),
			m.title(),
			m.scanStats(),
			m.treeView(),
		))
}

func (m *Model) handleEvent(event any) {
	log.Printf("### event#1 %T: locations %d", event, len(m.locations))
	defer func() {
		log.Printf("### event#2 %T: locations %d", event, len(m.locations))
	}()

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
			log.Println("### handleEvent: locations", len(m.locations))
		}

	case ui.KeyEvent:
		if location := m.currentLocation(); location != nil {
			m.handleArchiveKeyEvent(event, location)
		}

	case ui.MouseEvent:
		log.Printf("EventMouse: [%d:%d]", event.Col, event.Line)

	default:
		log.Printf("### unhandled event %#v", event)
	}
}

func (m *Model) analizeArchives() {
	log.Println("### analizeArchives#1: locations", len(m.locations))
	defer func() {
		log.Println("### analizeArchives#3: locations", len(m.locations))
	}()

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
	log.Println("### analizeArchives#2: locations", len(m.locations))
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
		File: &File{Kind: Folder},
	}}

	log.Println("### buildFileTree#1: locations", len(m.locations))
	defer log.Println("### buildFileTree#2: locations", len(m.locations))

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
			current := m.locations[0].File
			fileStack := []*File{current}
			if info == nil {
				continue
			}
			if i > 0 && infos[0] != nil && infos[0].Hash == info.Hash {
				continue
			}
			if i == 0 {
				current.Size += info.Size
			}
			for _, dir := range path {
				sub := subFolder(current, dir)
				if i == 0 {
					sub.Size += info.Size
				}
				current = sub
				fileStack = append(fileStack, current)
			}

			status := Identical
			if i == 0 {
				for _, links := range m.links {
					if links.sourceLinks[info] == nil {
						status = SourceOnly
					}
				}
			} else {
				if i > 0 && infos[0] != nil {
					status = Discrepancy
				} else {
					status = CopyOnly
				}
			}

			currentFile := &File{
				Info:   info,
				Kind:   RegularFile,
				Status: status,
				Name:   name,
				Size:   info.Size,
			}
			current.Files = append(current.Files, currentFile)
			for _, current = range fileStack {
				current.Status = status.Merge(current.Status)
			}
		}
	}
	// printArchive(m.currentLocation().File, "")
}

func subFolder(dir *File, name string) *File {
	for i := range dir.Files {
		if name == dir.Files[i].Name && dir.Files[i].Kind == Folder {
			return dir.Files[i]
		}
	}
	subFolder := &File{Kind: Folder, Name: name}
	dir.Files = append(dir.Files, subFolder)
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
	if archive.Kind == RegularFile {
		kind = "F"
	}
	if archive.Kind == RegularFile {
		log.Printf("%s%s: %s status=%v size=%v hash=%v", prefix, kind, archive.Name, archive.Status, archive.Size, archive.Info.Hash)
	} else {
		log.Printf("%s%s: %s status=%v size=%v", prefix, kind, archive.Name, archive.Status, archive.Size)
	}
	for _, file := range archive.Files {
		printArchive(file, prefix+"│ ")
	}
}

func (m *Model) handleArchiveKeyEvent(key ui.KeyEvent, loc *location) {
	switch key.Name {
	case "Home":
		loc.Selected = loc.File.Files[0]

	case "End":
		loc.Selected = loc.File.Files[len(loc.File.Files)-1]

	case "PgUp":
		log.Printf("PgUp#1: loc.LineOffset=%v  m.ArchiveViewLines=%v", loc.LineOffset, m.archiveViewLines)
		if loc.LineOffset < m.archiveViewLines {
			loc.LineOffset = 0
			loc.Selected = loc.File.Files[0]
		} else {
			loc.LineOffset -= m.archiveViewLines
			loc.Selected = loc.File.Files[loc.LineOffset]
		}
		log.Printf("PgUp#2: loc.LineOffset=%v  m.ArchiveViewLines=%v", loc.LineOffset, m.archiveViewLines)

	case "PgDn":

	case "Up":
		if loc.Selected != nil {
			for i, file := range loc.File.Files {
				if file == loc.Selected && i > 0 {
					loc.Selected = loc.File.Files[i-1]
					break
				}
			}
		} else {
			loc.Selected = loc.File.Files[len(loc.File.Files)-1]
		}

	case "Down":
		if loc.Selected != nil {
			for i, file := range loc.File.Files {
				if file == loc.Selected && i+1 < len(loc.File.Files) {
					loc.Selected = loc.File.Files[i+1]
					break
				}
			}
		} else {
			loc.Selected = loc.File.Files[0]
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
	log.Println("### treeView: locations", len(m.locations))
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
				if location.LineOffset > ui.Y(len(location.File.Files)+1-int(height)) {
					location.LineOffset = ui.Y(len(location.File.Files) + 1 - int(height))
				}
				if location.LineOffset < 0 {
					location.LineOffset = 0
				}
				if location.Selected != nil {
					idx := ui.Y(-1)
					for i := range location.File.Files {
						if location.Selected == location.File.Files[i] {
							idx = ui.Y(i)
							break
						}
					}
					if idx >= 0 {
						if location.LineOffset > idx {
							location.LineOffset = idx
						}
						if location.LineOffset < idx+1-height {
							location.LineOffset = idx + 1 - height
						}
					}
				}
				rows := make([]ui.Widget, height)
				i := 0
				var file *File
				for i, file = range location.File.Files[location.LineOffset:] {
					if i >= len(rows) {
						break
					}
					if file.Kind == RegularFile {
						rows[i] = ui.Styled(styleFile(file.Status, location.Selected == file),
							ui.Row(
								ui.Text(file.Status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.Name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text(file.Info.ModTime.Format(time.DateTime), 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.Size), 18, 0),
							),
						)
					} else {
						rows[i] = ui.Styled(styleFolder(file.Status, location.Selected == file),
							ui.Row(
								ui.Text(file.Status.String(), 7, 0),
								ui.Text("  ", 2, 0),
								ui.Text(file.Name, 20, 1),
								ui.Text("  ", 2, 0),
								ui.Text("<Каталог>", 19, 0),
								ui.Text("  ", 2, 0),
								ui.Text(formatSize(file.Size), 18, 0),
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
