package model

import (
	"fmt"
	"path/filepath"
	"strings"
)

type AnalizeArchives struct{}

type maps struct {
	byName groupByName
	byHash groupByHash
}

type groupByName map[string]*FileMeta
type groupByHash map[string]FileMetas

type links struct {
	sourceLinks  map[*FileMeta]*FileMeta
	reverseLinks map[*FileMeta]*FileMeta
}

type analyzer struct {
	maps  []maps   // source, copy1, copy2, ...
	links []*links // copy1, copy2, ...
}

func (e AnalizeArchives) HandleEvent(m *Model) {
	for i := range m.Archives {
		if m.Archives[i].Files == nil {
			return
		}
	}
	a := newAnalyzer(m)
	a.buildFileTree(m)
}

func newAnalyzer(m *Model) *analyzer {
	analyzer := &analyzer{}
	analyzer.maps = make([]maps, len(m.Archives))
	for i := range m.Archives {
		analyzer.maps[i] = maps{
			byName: byName(m.Archives[i].Files),
			byHash: byHash(m.Archives[i].Files),
		}
	}

	analyzer.links = make([]*links, len(m.Archives)-1)
	for i, archive := range m.Archives[1:] {
		analyzer.links[i] = analyzer.linkArchives(archive.Files)
	}
	return analyzer
}

func (a *analyzer) linkArchives(copyMetas FileMetas) *links {
	result := &links{
		sourceLinks:  map[*FileMeta]*FileMeta{},
		reverseLinks: map[*FileMeta]*FileMeta{},
	}
	for _, copy := range copyMetas {
		if sources, ok := a.maps[0].byHash[copy.Hash]; ok {
			match(sources, copy, result.sourceLinks)
		}
	}

	for source, copy := range result.sourceLinks {
		result.reverseLinks[copy] = source
	}

	return result
}

// TODO: make comparison by path a priority over comparison by name
func match(sources FileMetas, copy *FileMeta, sourceMap map[*FileMeta]*FileMeta) *FileMeta {
	for _, source := range sources {
		if copy.FullName == source.FullName {
			sourceMap[source] = copy
			return nil
		}
	}

	for _, source := range sources {
		tmpCopy := sourceMap[source]
		sourceBase := filepath.Base(source.FullName)
		if filepath.Base(copy.FullName) == sourceBase && (tmpCopy == nil || filepath.Base(tmpCopy.FullName) != sourceBase) {
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
		sourceBase := filepath.Base(source.FullName)
		sourceDir := filepath.Dir(source.FullName)
		if filepath.Dir(copy.FullName) == sourceDir &&
			(tmpCopy == nil ||
				(filepath.Base(tmpCopy.FullName) != sourceBase && filepath.Dir(tmpCopy.FullName) != sourceDir)) {

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

func byName(metas FileMetas) groupByName {
	result := groupByName{}
	for _, meta := range metas {
		result[meta.FullName] = meta
	}
	return result
}

func byHash(metas FileMetas) groupByHash {
	result := groupByHash{}
	for _, meta := range metas {
		result[meta.Hash] = append(result[meta.Hash], meta)
	}
	return result
}

func (a *analyzer) buildFileTree(m *Model) {
	m.Root = &FileInfo{FileMeta: &FileMeta{FullName: "/"}, Name: "Архив"}
	m.Breadcrumbs = []Folder{{File: m.Root}}
	hashes := map[string][]int{}
	for idx, arch := range m.Archives {
		for _, file := range arch.Files {
			hash, ok := hashes[file.Hash]
			if !ok {
				hash = make([]int, len(m.Archives))
				hashes[file.Hash] = hash
			}
			hash[idx]++
		}
	}

	for archIdx, archive := range m.Archives {
		for _, file := range archive.Files {
			hash := hashes[file.Hash]
			path := strings.Split(file.FullName, "/")
			name := path[len(path)-1]
			path = path[:len(path)-1]

			current := m.Root
			fileStack := FileInfos{current}
			if archIdx > 0 {
				if _, ok := a.links[archIdx-1].reverseLinks[file]; ok {
					continue
				}
			}
			if archIdx == 0 {
				current.Size += file.Size
			}
			for _, folder := range path {
				sub := subFolder(current, folder)
				if archIdx == 0 {
					sub.Size += file.Size
				}
				if sub.Archive == "" {
					sub.Archive = file.Archive
				}
				if sub.ModTime.Before(file.ModTime) {
					sub.ModTime = file.ModTime
				}
				current = sub
				fileStack = append(fileStack, current)
			}

			status := Identical
			for _, h := range hash[1:] {
				if hash[0] > h {
					status = SourceOnly
				}
			}
			if hash[0] == 0 {
				status = CopyOnly
			}

			currentFile := &FileInfo{
				FileMeta: file,
				Name:     name,
				Kind:     FileRegular,
				Status:   status,
			}
			current.Files = append(current.Files, currentFile)
			for _, current = range fileStack {
				current.Status = status.Merge(current.Status)
			}
		}
	}
	m.Sort()
}

func subFolder(folder *FileInfo, name string) *FileInfo {
	for i := range folder.Files {
		if name == folder.Files[i].Name && folder.Files[i].Kind == FileFolder {
			return folder.Files[i]
		}
	}
	subFolder := &FileInfo{
		FileMeta: &FileMeta{
			Archive:  folder.Archive,
			FullName: name,
		},
		Name: name,
		Kind: FileFolder,
	}
	folder.Files = append(folder.Files, subFolder)
	return subFolder
}

func (a *links) String() string {
	b := &strings.Builder{}
	fmt.Fprintln(b, "Source Map:")
	for s, c := range a.sourceLinks {
		fmt.Fprintf(b, "  %s -> %s %s\n", s.FullName, c.FullName, s.Hash)
	}
	fmt.Fprintln(b, "Reverse Map:")
	for s, c := range a.reverseLinks {
		fmt.Fprintf(b, "  %s -> %s %s\n", s.FullName, c.FullName, s.Hash)
	}
	return b.String()
}
