package ui

import (
	"arch/device"
	"arch/files"
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"
)

type model struct {
	paths            []string
	scanStates       []*files.ScanState
	locations        []location
	scanResults      []*files.ArchiveInfo
	maps             []maps   // source, copy1, copy2, ...
	links            []*links // copy1, copy2, ...
	screenSize       Size
	archiveViewLines int
	ctx              *Context
	lastMouseEvent   device.MouseEvent
	sortColumn       sortColumn
	sortAscending    []bool
}

type fileInfo struct {
	kind    fileKind
	status  fileStatus
	archive string
	path    string
	name    string
	size    int
	modTime time.Time
	hash    string
	files   []*fileInfo
}

type location struct {
	file       *fileInfo
	selected   *fileInfo
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
type groupByHash map[string][]*files.FileInfo

type selectFile *fileInfo
type selectFolder *fileInfo

func (s fileStatus) Merge(other fileStatus) fileStatus {
	if s > other {
		return s
	}
	return other
}

func (m *model) handleFilesEvent(event files.Event) {
	log.Printf("### fs event %#v", event)
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
		log.Println("handleFilesEvent: doneScanning = ", doneScanning)
		if doneScanning {
			m.analizeArchives()
		}

	case files.ScanError:
		log.Printf("FS Error: Archive %q", event.Archive)
		log.Printf("FS Error: Path %q", event.Path)
		log.Printf("FS Error: Name %v", event.Error)
		log.Panicf("### unhandled files event %#v", event)

	default:
		log.Panicf("### unhandled files event %#v", event)
	}
}

func (m *model) analizeArchives() {
	log.Println("analizeArchives")
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

func byName(infos []*files.FileInfo) groupByName {
	result := groupByName{}
	for _, info := range infos {
		result[info.Name] = info
	}
	return result
}

func byHash(archive []*files.FileInfo) groupByHash {
	result := groupByHash{}
	for _, info := range archive {
		result[info.Hash] = append(result[info.Hash], info)
	}
	return result
}

func (m *model) buildFileTree() {
	m.locations = []location{{
		file: &fileInfo{name: " Архив", kind: folder},
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
		for i, maps := range m.maps {
			infos[i] = maps.byName[fullName]
		}
		for i, info := range infos {
			current := m.locations[0].file
			fileStack := []*fileInfo{current}
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
				if sub.archive == "" {
					sub.archive = info.Archive
					sub.path = filepath.Join(path[:pathIdx]...)
				}
				if sub.modTime.Before(info.ModTime) {
					sub.modTime = info.ModTime
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
				if infos[0] != nil {
					status = discrepancy
				} else if m.maps[0].byHash[info.Hash] != nil {
					status = extraCopy
				} else {
					status = copyOnly
				}
			}

			currentFile := &fileInfo{
				kind:    regularFile,
				status:  status,
				archive: info.Archive,
				path:    filepath.Dir(info.Name),
				name:    name,
				size:    info.Size,
				modTime: info.ModTime,
				hash:    info.Hash,
			}
			current.files = append(current.files, currentFile)
			for _, current = range fileStack {
				current.status = status.Merge(current.status)
			}
		}
	}
	m.sort()
	PrintArchive(m.currentLocation().file, "")
}

func subFolder(dir *fileInfo, name string) *fileInfo {
	for i := range dir.files {
		if name == dir.files[i].name && dir.files[i].kind == folder {
			return dir.files[i]
		}
	}
	subFolder := &fileInfo{kind: folder, name: name}
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

func (m *model) linkArchives(copyInfos []*files.FileInfo) *links {
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

func match(sources []*files.FileInfo, copy *files.FileInfo, sourceMap map[*files.FileInfo]*files.FileInfo) *files.FileInfo {
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

func PrintArchive(archive *fileInfo, prefix string) {
	kind := "D"
	if archive.kind == regularFile {
		kind = "F"
	}
	if archive.kind == regularFile {
		log.Printf("%s%s: %s status=%v size=%v hash=%v", prefix, kind, archive.name, archive.status, archive.size, archive.hash)
	} else {
		log.Printf("%s%s: %s status=%v size=%v", prefix, kind, archive.name, archive.status, archive.size)
	}
	for _, file := range archive.files {
		PrintArchive(file, prefix+"│ ")
	}
}
