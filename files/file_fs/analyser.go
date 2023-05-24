package file_fs

import (
	"arch/model"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

type maps struct {
	byName groupByName
	byHash groupByHash
}

type groupByName map[string]*model.FileMeta
type groupByHash map[string]model.FileMetas

type links struct {
	sourceLinks  map[*model.FileMeta]*model.FileMeta
	reverseLinks map[*model.FileMeta]*model.FileMeta
}

type analyzer struct {
	maps  []maps   // source, copy1, copy2, ...
	links []*links // copy1, copy2, ...
}

func analizeArchives(m *model.Model) {
	log.Println("analizeArchives")
	a := newAnalyzer(m)
	a.buildFileTree(m)
}

func newAnalyzer(m *model.Model) *analyzer {
	analyzer := &analyzer{}
	for i := range m.Archives {
		m.Archives[i].ScanState = nil
	}
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

func (a *analyzer) linkArchives(copyInfos model.FileMetas) *links {
	result := &links{
		sourceLinks:  map[*model.FileMeta]*model.FileMeta{},
		reverseLinks: map[*model.FileMeta]*model.FileMeta{},
	}
	for _, copy := range copyInfos {
		if sources, ok := a.maps[0].byHash[copy.Hash]; ok {
			match(sources, copy, result.sourceLinks)
		}
	}

	for source, copy := range result.sourceLinks {
		result.reverseLinks[copy] = source
	}

	return result
}

func match(sources model.FileMetas, copy *model.FileMeta, sourceMap map[*model.FileMeta]*model.FileMeta) *model.FileMeta {
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

func byName(metas model.FileMetas) groupByName {
	result := groupByName{}
	for _, meta := range metas {
		result[meta.FullName] = meta
	}
	return result
}

func byHash(metas model.FileMetas) groupByHash {
	result := groupByHash{}
	for _, meta := range metas {
		result[meta.Hash] = append(result[meta.Hash], meta)
	}
	return result
}

func (a *analyzer) buildFileTree(m *model.Model) {
	hashes := map[string][]int{}
	for idx, arch := range m.Archives {
		for _, file := range arch.Files {
			hash, ok := hashes[file.Hash]
			if !ok {
				hashes[file.Hash] = make([]int, len(m.Archives))
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
			fileStack := model.FileInfos{current}
			if archIdx > 0 && hash[0] > 0 && hash[0] == hash[archIdx] {
				continue
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
					sub.FullName = name
				}
				if sub.ModTime.Before(file.ModTime) {
					sub.ModTime = file.ModTime
				}
				current = sub
				fileStack = append(fileStack, current)
			}

			status := model.Identical
			for _, h := range hash[1:] {
				if hash[0] > h {
					status = model.SourceOnly
				}
			}
			if hash[0] == 0 {
				status = model.CopyOnly
			}

			currentFile := &model.FileInfo{
				FileMeta: file,
				Name:     name,
				Kind:     model.FileRegular,
				Status:   status,
			}
			current.Files = append(current.Files, currentFile)
			for _, current = range fileStack {
				current.Status = status.Merge(current.Status)
			}
		}
	}
}

func subFolder(folder *model.FileInfo, name string) *model.FileInfo {
	for i := range folder.Files {
		if name == folder.Files[i].FullName && folder.Files[i].Kind == model.FileFolder {
			return folder.Files[i]
		}
	}
	subFolder := &model.FileInfo{
		FileMeta: &model.FileMeta{
			Archive:  folder.Archive,
			FullName: name,
		},
		Kind: model.FileFolder,
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
