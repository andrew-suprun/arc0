package app

import (
	"arch/files"
	"arch/ui"
	"arch/view"
)

type app struct {
	fs       files.FS
	model    *view.Model
	renderer ui.Renderer
	width    ui.X
	height   ui.Y
	quit     bool
}

func Run(paths []string, fs files.FS, renderer ui.Renderer) {
	app := &app{
		fs:       fs,
		renderer: renderer,
		model:    view.NewModel(paths),
	}

	events := make(chan any)

	go func() {
		for {
			events <- app.renderer.PollEvent()
		}
	}()

	for _, archive := range paths {
		go func(archive string) {
			for ev := range fs.Scan(archive) {
				events <- ev
			}
		}(archive)
	}

	for !app.quit {
		app.render(<-events)
	}
	app.fs.Stop()
	app.renderer.Exit()
}

func (app *app) render(event any) {
	sync := false
	switch event := event.(type) {
	case ui.KeyEvent:
		if event.Name == "Ctrl+C" {
			app.quit = true
			return
		}

	case ui.ResizeEvent:
		app.width, app.height = ui.X(event.Width), ui.Y(event.Height)
		sync = true
	}

	screen := app.model.View(event)
	screen.Render(app.renderer, 0, 0, app.width, app.height, view.DefaultStyle)
	if sync {
		app.renderer.Sync()
		sync = false
	} else {
		app.renderer.Show()
	}
}
