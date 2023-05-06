package tcell

import (
	"arch/ui"
	"log"

	"github.com/gdamore/tcell/v2"
)

type renderer struct {
	screen tcell.Screen
}

func NewRenderer() (ui.Renderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.SetStyle(noStyle)
	screen.EnableMouse()
	screen.EnablePaste()

	return &renderer{screen: screen}, nil
}

func (r *renderer) Text(runes []rune, x ui.X, y ui.Y, style ui.Style) {
	for i, rune := range runes {
		r.screen.SetContent(int(x)+i, int(y), rune, nil, tcellStyle(style))
	}
}

func (r *renderer) MouseTarget(command any, x ui.X, y ui.Y) {
	// TODO
}

func (r *renderer) ScrollTarget(command any, x ui.X, y ui.Y) {
	// TODO
}

func (r *renderer) PollEvent() any {
	ev := r.screen.PollEvent()
	for {
		if ev, mouseEvent := ev.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
			break
		}
		ev = r.screen.PollEvent()
	}
	return r.uiEvent(ev)
}

func (r *renderer) Show() {
	r.screen.Show()
}

func (r *renderer) Sync() {
	r.screen.Sync()
}

func (r *renderer) Exit() {
	r.screen.Fini()
}

func (r *renderer) uiEvent(ev tcell.Event) any {
	log.Printf("### tcell.Event: %#v\n", ev)

	// TODO: temporary
	switch ev := ev.(type) {
	case *tcell.EventResize:
		w, h := ev.Size()
		return ui.ResizeEvent{Width: w, Height: h}

	case *tcell.EventKey:
		return ui.KeyEvent{Name: ev.Name(), Rune: ev.Rune()}

	case *tcell.EventMouse:
		x, y := ev.Position()
		return ui.MouseEvent{Col: x, Line: y}

	default:
		return nil
	}
}

var (
	noStyle            = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)
	styleDefault       = tcell.StyleDefault.Foreground(tcell.PaletteColor(231)).Background(tcell.PaletteColor(18))
	styleFolder        = tcell.StyleDefault.Foreground(tcell.PaletteColor(230)).Background(tcell.PaletteColor(18)).Bold(true)
	styleFile          = tcell.StyleDefault.Foreground(tcell.PaletteColor(231)).Background(tcell.PaletteColor(18))
	styleAppName       = tcell.StyleDefault.Foreground(tcell.PaletteColor(226)).Background(tcell.PaletteColor(0)).Bold(true).Italic(true)
	styleArchiveName   = tcell.StyleDefault.Foreground(tcell.PaletteColor(231)).Background(tcell.PaletteColor(0)).Bold(true)
	styleArchiveHeader = tcell.StyleDefault.Foreground(tcell.PaletteColor(231)).Background(tcell.PaletteColor(8)).Bold(true)
	styleProgressBar   = tcell.StyleDefault.Foreground(tcell.PaletteColor(231)).Background(tcell.PaletteColor(20))
)

func tcellStyle(uiStyle ui.Style) tcell.Style {
	switch uiStyle {
	case ui.NoStyle:
		return noStyle
	case ui.StyleDefault:
		return styleDefault
	case ui.StyleHeader:
		return styleDefault
	case ui.StyleAppTitle:
		return styleAppName
	case ui.StyleArchiveName:
		return styleArchiveName
	case ui.StyleArchiveHeader:
		return styleArchiveHeader
	case ui.StyleFile:
		return styleFile
	case ui.StyleFolder:
		return styleFolder
	case ui.StyleProgressBar:
		return styleProgressBar
	default:
		return noStyle
	}
}
