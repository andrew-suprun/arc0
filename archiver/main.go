package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"

	"scanner/fs"
	"scanner/lifecycle"
)

func main() {
	lc := lifecycle.New()

	a := app.New()
	a.Lifecycle().SetOnStopped(func() {
		lc.Stop()
	})
	w := a.NewWindow("List Widget")
	w.Resize(fyne.NewSize(4000, 3000))

	// vbox := container.NewVBox()
	// vbox.Add()

	fileProgress := widget.NewProgressBar()
	fileProgress.Min = 0.0
	fileProgress.Max = 100.0
	// fileProgress.TextFormatter = func() string {
	// 	return fmt.Sprintf("%.1f%%", fileProgress.Value)
	// }

	overallProgress := widget.NewProgressBar()
	overallProgress.Min = 0.0
	overallProgress.Max = 100.0
	overallProgress.TextFormatter = func() string {
		return fmt.Sprintf("%.1f%%", overallProgress.Value)
	}

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
		overallProgress,
	)

	card := widget.NewCard("/Volumes/Seagate/tmp", "", form)

	border := container.NewBorder(nil, nil, nil, nil, card)
	w.SetContent(border)

	// ui := makeHashUI()
	go hash(lc, form, "/Volumes/Seagate/tmp")

	w.ShowAndRun()
}

func hash(lc *lifecycle.Lifecycle, form *fyne.Container, path string) {
	results := fs.Scan(lc, path)
	var start time.Time
	var nilTime time.Time
	for result := range results {
		if start == nilTime {
			start = time.Now()
		}
		switch update := result.(type) {
		case fs.ScanFileResult:
		case fs.ScanStat:
			fileProgress := float64(update.Hashed) / float64(update.Size)
			etaProgress := float64(update.TotalHashed) / float64(update.TotalToHash)
			overallHashed := update.TotalSize - update.TotalToHash + update.TotalHashed
			overalProgress := float64(overallHashed) / float64(update.TotalSize)
			dur := time.Since(start)
			eta := start.Add(time.Duration(float64(dur) / etaProgress))
			remainig := time.Until(eta)
			form.Objects[1].(*widget.Label).Text = update.Path
			form.Objects[3].(*widget.Label).Text = eta.Format(time.TimeOnly)
			form.Objects[5].(*widget.Label).Text = remainig.Truncate(time.Second).String()
			form.Objects[7].(*widget.ProgressBar).Value = fileProgress * 100
			form.Objects[9].(*widget.ProgressBar).Value = overalProgress * 100
			form.Refresh()

		case fs.ScanError:
			log.Printf("stat: file=%s error=%#v, %#v\n", update.Path, update.Error, errors.Unwrap(update.Error))
		}
	}

	form.Objects[1].(*widget.Label).Text = ""
	form.Objects[3].(*widget.Label).Text = "Done"
	form.Objects[5].(*widget.ProgressBar).Value = 100
	form.Objects[7].(*widget.ProgressBar).Value = 100
	form.Refresh()
}
