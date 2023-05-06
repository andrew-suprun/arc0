package app

import (
	"arch/files"
	"arch/ui"
	"arch/view"
	"fmt"
	"log"
	"path/filepath"
	"strings"
)

type app struct {
	paths         []string
	fs            files.FS
	scanResults   files.ArchiveInfos
	maps          []maps   // source, copy1, copy2, ...
	links         []*links // copy1, copy2, ...
	model         view.Model
	renderer      ui.Renderer
	width, height int
	quit          bool
}

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

func Run(paths []string, fs files.FS, renderer ui.Renderer) {
	app := &app{
		paths:       paths,
		fs:          fs,
		renderer:    renderer,
		scanResults: make(files.ArchiveInfos, len(paths)),
		model:       view.Model{ScanStates: make([]*files.ScanState, len(paths))},
	}

	tcellChan := make(chan any)

	go func() {
		for {
			tcellChan <- app.renderer.PollEvent()
		}
	}()

	fsInput := make(chan any)

	for _, archive := range paths {
		go func(archive string) {
			for ev := range fs.Scan(archive) {
				fsInput <- ev
			}
		}(archive)
	}

	for !app.quit {
		select {
		case event := <-fsInput:
			app.handleFsEvent(event)
		case event := <-tcellChan:
			app.handleUiEvent(event)
		}
		app.render()
	}
	app.fs.Stop()
	app.renderer.Exit()
}

func (app *app) analizeArchives() {
	app.maps = make([]maps, len(app.scanResults))
	for i, scan := range app.scanResults {
		app.maps[i] = maps{
			byName: byName(scan.Files),
			byHash: byHash(scan.Files),
		}
	}

	app.links = make([]*links, len(app.scanResults)-1)
	for i, copy := range app.scanResults[1:] {
		app.links[i] = app.linkArchives(copy.Files)
	}
	app.buildFileTree()
	app.model.Location.File = app.model.Root
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

func (app *app) buildFileTree() {
	app.model.Root = &view.File{Kind: view.Folder}
	uniqueFileNames := map[string]struct{}{}
	for _, info := range app.scanResults[0].Files {
		uniqueFileNames[info.Name] = struct{}{}
	}
	for i, copyScan := range app.scanResults[1:] {
		reverseLinks := app.links[i].reverseLinks
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
		infos := make([]*files.FileInfo, len(app.maps))
		for i, info := range app.maps {
			infos[i] = info.byName[fullName]
		}
		for i, info := range infos {
			current := app.model.Root
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
			}

			status := view.Identical
			if i == 0 {
				for _, links := range app.links {
					if links.sourceLinks[info] == nil {
						status = view.SourceOnly
					}
				}
			} else {
				if i > 0 && infos[0] != nil {
					status = view.Discrepancy
				} else {
					status = view.CopyOnly
				}
			}

			currentFile := &view.File{
				Parent: current,
				Info:   info,
				Kind:   view.RegularFile,
				Status: status,
				Name:   name,
				Size:   info.Size,
			}
			current.SubFolders = append(current.SubFolders, currentFile)
			for current != nil {
				current.Status = status.Merge(current.Status)
				current = current.Parent
			}
		}
	}
	printArchive(app.model.Root, "")
}

func subFolder(dir *view.File, name string) *view.File {
	for i := range dir.SubFolders {
		if name == dir.SubFolders[i].Name {
			return dir.SubFolders[i]
		}
	}
	subFolder := &view.File{Parent: dir, Kind: view.Folder, Name: name}
	dir.SubFolders = append(dir.SubFolders, subFolder)
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

func (app *app) linkArchives(copyInfos files.FileInfos) *links {
	result := &links{
		sourceLinks:  map[*files.FileInfo]*files.FileInfo{},
		reverseLinks: map[*files.FileInfo]*files.FileInfo{},
	}
	for _, copy := range copyInfos {
		if sources, ok := app.maps[0].byHash[copy.Hash]; ok {
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
			log.Printf("    same name: %s -> %s", source.Name, copy.Name)
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

func printArchive(archive *view.File, prefix string) {
	kind := "D"
	if archive.Kind == view.RegularFile {
		kind = "F"
	}
	if archive.Kind == view.RegularFile {
		log.Printf("%s%s: %s status=%v size=%v hash=%v", prefix, kind, archive.Name, archive.Status, archive.Size, archive.Info.Hash)
	} else {
		log.Printf("%s%s: %s status=%v size=%v", prefix, kind, archive.Name, archive.Status, archive.Size)
	}
	for _, file := range archive.SubFolders {
		printArchive(file, prefix+"│ ")
	}
}

func (app *app) handleFsEvent(event any) {
	switch event := event.(type) {
	case *files.ScanState:
		for i := range app.paths {
			if app.paths[i] == event.Archive {
				app.model.ScanStates[i] = event
				break
			}
		}

	case *files.ArchiveInfo:
		for i := range app.paths {
			if app.paths[i] == event.Archive {
				app.model.ScanStates[i] = nil
				app.scanResults[i] = event
				break
			}
		}
		doneScanning := true
		for i := range app.paths {
			if app.scanResults[i] == nil {
				doneScanning = false
				break
			}
		}
		if doneScanning {
			app.analizeArchives()
		}

	default:
		log.Printf("### unhandled event %#v", event)
	}
}

func (app *app) handleUiEvent(event any) {
	switch ev := event.(type) {
	case ui.ResizeEvent:
		log.Printf("ResizeEvent: [%d:%d]", ev.Width, ev.Height)
		app.width, app.height = ev.Width, ev.Height
		app.render()
		app.renderer.Sync()
	case ui.KeyEvent:
		log.Printf("EventKey: name=%s '%c'", ev.Name, ev.Rune)
		if ev.Name == "Ctrl+C" {
			app.quit = true
		}

	case ui.MouseEvent:
		log.Printf("EventMouse: [%d:%d]", ev.Col, ev.Line)
	default:
	}
}

func (app *app) render() {
	// TODO
	screen := app.model.View()
	screen.Render(app.renderer, ui.X(0), ui.Y(0), ui.W(app.width), ui.H(app.height), ui.StyleDefault)
	app.renderer.Show()
}

// 	for len(contents) < app.height-2 {
// 		contents = append(contents, ui.Line(ui.Text(""), ui.Text(""), ui.Text("")))
// 	}

// 	view := ui.View(0, 1, app.width, app.height-2,
// 		ui.Styled(
// 			ui.StyleDefault,
// 			ui.Layout(ui.Fields{ui.Pad(" "), ui.Fixed(26), ui.Pad(" "), ui.Flex(1), ui.Pad(" ")}, contents...)),
// 	)
// 	app.renderer.Render(view...)
// }

// func (app *app) drawArchive() {
// 	if app.root == nil {
// 		return
// 	}

// 	// idx := app.location.lineOffset

// 	contents := []ui.Widget{

// 		ui.Layout(ui.Fields{ui.Pad(" "), ui.Fixed(6), ui.Pad(" "), ui.Flex(1), ui.Pad(" "), ui.Fixed(19), ui.Pad(" "), ui.Fixed(20), ui.Pad(" ")},
// 			ui.Styled(ui.StyleArchiveHeader,
// 				ui.Line(ui.Text("Статус"), ui.Text("Документ"), ui.Text("Время Изменения"), ui.RText("Размер")),
// 			),
// 		),
// 	}

// 	// archive := app.archives[app.archiveIdx]
// 	// location := app.locations[app.archiveIdx]
// 	// for _, dir := range location.path {
// 	// 	archive = archive.subFolders[dir]
// 	// }

// 	// // subfolders
// 	// b.SetStyle(ui.StyleFolder)
// 	// subFolders := make([]folder, 0, len(archive.subFolders))
// 	// for name, folder := range archive.subFolders {
// 	// 	folder.name = name
// 	// 	subFolders = append(subFolders, folder)
// 	// }
// 	// sort.Slice(subFolders, func(i, j int) bool {
// 	// 	return subFolders[i].name < subFolders[j].name
// 	// })
// 	// for _, subFolder := range subFolders {
// 	// 	b.AddText("")
// 	// 	b.AddText(" " + subFolder.name)
// 	// 	b.AddText(" Каталог")
// 	// 	b.AddText(formatSize(subFolder.size))
// 	// 	b.LayoutLine()

// 	// 	if app.lineOffset >= app.height-1 {
// 	// 		break
// 	// 	}
// 	// }

// 	// // files
// 	// b.SetStyle(ui.StyleFile)
// 	// files := make([]file, 0, len(archive.files))
// 	// for name, file := range archive.files {
// 	// 	file.name = name
// 	// 	files = append(files, file)
// 	// }
// 	// sort.Slice(files, func(i, j int) bool {
// 	// 	return files[i].name < files[j].name
// 	// })
// 	// for _, file := range files {
// 	// 	b.AddText("")
// 	// 	b.AddText(" " + file.name)
// 	// 	b.AddText(" " + file.modTime.Format(time.DateTime))
// 	// 	b.AddText(formatSize(file.size))
// 	// 	b.LayoutLine()

// 	// 	if app.lineOffset >= app.height-1 {
// 	// 		break
// 	// 	}
// 	// }

// 	view := ui.View(0, 1, app.width, app.height-2,
// 		ui.Styled(ui.StyleDefault,
// 			ui.Layout(ui.Fields{ui.Pad(" "), ui.Fixed(7), ui.Pad(" "), ui.Flex(1), ui.Pad(" "), ui.Fixed(19), ui.Pad(" "), ui.Fixed(16), ui.Pad(" ")},
// 				contents...,
// 			),
// 		),
// 	)
// 	app.renderer.Render(view...)
// }

// func (app *app) drawStatusLine() {
// 	view := ui.View(0, app.height-1, app.width, 1,
// 		ui.Styled(ui.StyleArchiveName,
// 			ui.Layout(ui.Fields{ui.Pad(" "), ui.Flex(1), ui.Pad(" ")},
// 				ui.Line(ui.Text("Status line will be here...")),
// 			),
// 		),
// 	)
// 	app.renderer.Render(view...)
// }

// func formatSize(size int) string {
// 	str := fmt.Sprintf("%13d ", size)
// 	slice := []string{str[:1], str[1:4], str[4:7], str[7:10]}
// 	b := strings.Builder{}
// 	for _, s := range slice {
// 		b.WriteString(s)
// 		if s == " " || s == "   " {
// 			b.WriteString(" ")
// 		} else {
// 			b.WriteString(",")
// 		}
// 	}
// 	b.WriteString(str[10:])
// 	return b.String()
// }
