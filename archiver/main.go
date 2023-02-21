package main

import (
	"fmt"
	"log"
	"strings"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"scanner/fs"
	"scanner/lifecycle"
)

type archiver struct {
	*lifecycle.Lifecycle
	fyne.App
	fyne.Window
	ch    chan any
	cards *fyne.Container
	data  chan *state
}

type state struct {
	source  scanUI
	targets []scanUI
}

type scanUI struct {
	path  string
	start time.Time
	form  *fyne.Container
}

func newArchiver() *archiver {
	lc := lifecycle.New()
	a := app.NewWithID("archiver")
	ch := make(chan any)
	a.Lifecycle().SetOnStopped(func() {
		lc.Stop()
	})
	w := a.NewWindow("Archiver")
	w.Resize(fyne.NewSize(4000, 750))
	cards := container.NewVBox()
	border := container.NewBorder(nil, nil, nil, nil, cards)
	w.SetContent(border)
	data := make(chan *state, 1)
	data <- &state{}

	return &archiver{
		Lifecycle: lc,
		App:       a,
		Window:    w,
		ch:        ch,
		cards:     cards,
		data:      data,
	}
}

func main() {
	app := newArchiver()
	go app.run()
	app.ch <- selectExistingSourceMsg{}
	app.ShowAndRun()
}

func (app *archiver) run() {
	for msg := range app.ch {
		go app.handleMsg(msg)
	}
}

type selectExistingSourceMsg struct{}
type selectNewSourceMsg struct{}
type selectTargetsMsg struct{}
type hashPathMsg struct {
	path string
}

func (app *archiver) handleMsg(msg any) {
	switch msg := msg.(type) {
	case selectExistingSourceMsg:
		app.selectExistingSource()
	case selectNewSourceMsg:
		app.selectNewSource()
	case selectTargetsMsg:
		app.selectTargets()
	case hashPathMsg:
		app.hashPath(msg.path)
	case fs.ScanStat:
		app.scanStat(msg)
	case fs.ScanFileResult:
		app.scanFileInfo(msg)
	default:
		log.Panicf("Unhandled msg: %#v\n", msg)
	}
}

func (app *archiver) selectExistingSource() {
	sourcesStr := app.Preferences().String("sources")
	fmt.Println(sourcesStr)
	if sourcesStr != "" {
		var done atomic.Bool
		sourceBtns := container.NewVBox()
		d := dialog.NewCustom("Select Source", "Cancel", sourceBtns, app.Window)
		sources := strings.Split(sourcesStr, "|")
		for i := range sources {
			source := sources[i]
			btn := widget.NewButton(source, func() {
				fmt.Println("### 1")
				form := scanForm()
				data := <-app.data
				data.source = scanUI{
					path: source,
					form: form,
				}
				app.data <- data
				card := widget.NewCard(source, "", form)
				app.cards.Add(card)
				app.ch <- hashPathMsg{path: source}
				app.ch <- selectTargetsMsg{}
				done.Store(true)
				d.Hide()
			})
			btn.Importance = widget.HighImportance
			sourceBtns.Add(btn)
		}
		anotherSourceBtn := widget.NewButton("Select Another Source", func() {
			fmt.Println("### 2")
			app.ch <- selectNewSourceMsg{}
			done.Store(true)
			d.Hide()
		})
		anotherSourceBtn.Importance = widget.HighImportance
		sourceBtns.Add(anotherSourceBtn)
		d.SetOnClosed(func() {
			if !done.Load() {
				fmt.Println("### 3")
				app.ch <- selectNewSourceMsg{}
			}
		})
		d.Show()
	} else {
		fmt.Println("### 4")
		app.ch <- selectNewSourceMsg{}
	}
}

func (app *archiver) selectNewSource() {
	dialog.ShowFolderOpen(func(url fyne.ListableURI, err error) {
		if url == nil {
			app.Quit()
		}

		path := url.Path()
		sourcesStr := app.Preferences().String("sources")
		sources := strings.Split(sourcesStr, "|")
		if sourcesStr == "" {
			app.Preferences().SetString("sources", path)
		} else {
			found := false
			for _, source := range sources {
				if path == source {
					found = true
					break
				}
			}
			if !found {
				app.Preferences().SetString("sources", sourcesStr+"|"+path)
			}
		}

		form := scanForm()
		data := <-app.data
		data.source = scanUI{
			path: path,
			form: form,
		}
		app.data <- data
		card := widget.NewCard(path, "", form)
		app.cards.Add(card)
		app.ch <- hashPathMsg{path: path}
		app.ch <- selectTargetsMsg{}
	}, app.Window)
}

func (app *archiver) selectTargets() {
	fmt.Println("select targets")
}

func scanForm() *fyne.Container {
	fileProgress := widget.NewProgressBar()
	fileProgress.Min = 0.0
	fileProgress.Max = 100.0

	form := container.New(layout.NewFormLayout(),
		widget.NewLabel("File"),
		widget.NewLabel("file name"),

		widget.NewLabel("ETA"),
		widget.NewLabel("file eta"),

		widget.NewLabel("Time Remaining"),
		widget.NewLabel("time remaining"),

		widget.NewLabel("File Progress"),
		fileProgress,

		widget.NewLabel("Overal Progress"),
		widget.NewProgressBarInfinite(),
	)

	return form
}

var nilTime time.Time

func (app *archiver) scanStat(update fs.ScanStat) {
	data := <-app.data
	defer func() {
		app.data <- data
	}()
	var info *scanUI
	if data.source.path == update.Base {
		info = &data.source
	}
	if info == nil {
		log.Panicf("Cannot find scan info for %v\n", update.Path)
	}
	if info.start == nilTime {
		overallProgress := widget.NewProgressBar()
		overallProgress.Min = 0.0
		overallProgress.Max = 100.0
		overallProgress.TextFormatter = func() string {
			return fmt.Sprintf("%.1f%%", overallProgress.Value)
		}
		info.form.Objects[9] = overallProgress

		info.start = time.Now()
	}

	fileProgress := float64(update.Hashed) / float64(update.Size)
	etaProgress := float64(update.TotalHashed) / float64(update.TotalToHash)
	overallHashed := update.TotalSize - update.TotalToHash + update.TotalHashed
	overallProgress := float64(overallHashed) / float64(update.TotalSize)
	dur := time.Since(info.start)
	eta := info.start.Add(time.Duration(float64(dur) / etaProgress))
	remainig := time.Until(eta)
	info.form.Objects[1].(*widget.Label).Text = update.Path
	info.form.Objects[3].(*widget.Label).Text = eta.Format(time.TimeOnly)
	info.form.Objects[5].(*widget.Label).Text = remainig.Truncate(time.Second).String()
	info.form.Objects[7].(*widget.ProgressBar).Value = fileProgress * 100
	if pb, ok := info.form.Objects[9].(*widget.ProgressBar); ok {
		pb.Value = overallProgress * 100
	}
	info.form.Refresh()
}

func (app *archiver) scanFileInfo(info fs.ScanFileResult) {
	fmt.Printf("file %v\n", info.Path)
}

func (app *archiver) hashPath(path string) {
	fs.Scan(app.Lifecycle, path, app.ch)

	// form.RemoveAll()
	// form.Add(widget.NewLabel("Done"))
	// form.Refresh()
}
