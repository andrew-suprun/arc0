package tcell

import (
	"arch/device"
	"arch/ui"
	"log"

	"github.com/gdamore/tcell/v2"
)

type tcellRenderer struct {
	screen tcell.Screen
}

func NewDevice() (device.Device, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()

	return &tcellRenderer{screen: screen}, nil
}

func (r *tcellRenderer) PollEvent() any {
	ev := r.screen.PollEvent()
	for {
		if ev, mouseEvent := ev.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
			break
		}
		ev = r.screen.PollEvent()
	}
	switch ev := ev.(type) {
	case *tcell.EventResize:
		w, h := ev.Size()
		return ui.ResizeEvent{Width: w, Height: h}

	case *tcell.EventKey:
		log.Printf("key: name=%v rune='%v' mod=%v", ev.Name(), ev.Rune(), ev.Modifiers())
		return ui.KeyEvent{Name: ev.Name(), Rune: ev.Rune()}

	case *tcell.EventMouse:
		x, y := ev.Position()
		return ui.MouseEvent{Col: x, Line: y}

	default:
		return nil
	}
}

func (r *tcellRenderer) Text(runes []rune, x, y int, style device.Style) {
	for i, rune := range runes {
		style := tcell.StyleDefault.
			Foreground(tcell.PaletteColor(int(style.FG))).
			Background(tcell.PaletteColor(int(style.BG))).
			Bold(style.Flags&device.Bold == device.Bold).
			Italic(style.Flags&device.Italic == device.Italic).
			Reverse(style.Flags&device.Reverse == device.Reverse)

		r.screen.SetContent(int(x)+i, int(y), rune, nil, style)
	}
}

func (r *tcellRenderer) Show() {
	r.screen.Show()
}

func (r *tcellRenderer) Sync() {
	r.screen.Sync()
}

func (r *tcellRenderer) Exit() {
	r.screen.Fini()
}
