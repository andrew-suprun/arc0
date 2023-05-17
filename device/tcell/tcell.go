package tcell

import (
	"arch/device"
	"arch/ui"
	"log"

	"github.com/gdamore/tcell/v2"
)

type tcellDevice struct {
	screen tcell.Screen
}

func NewDevice() (*tcellDevice, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()

	return &tcellDevice{screen: screen}, nil
}

func (r *tcellDevice) PollEvent() any {
	ev := r.screen.PollEvent()
	for {
		if ev, mouseEvent := ev.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
			break
		}
		ev = r.screen.PollEvent()
	}
	switch ev := ev.(type) {
	case *tcell.EventResize:
		r.screen.Sync()
		w, h := ev.Size()
		return ui.ResizeEvent{Width: w, Height: h}

	case *tcell.EventKey:
		log.Printf("key: name=%v rune='%v' mod=%v", ev.Name(), ev.Rune(), ev.Modifiers())
		return ui.KeyEvent{Name: ev.Name(), Rune: ev.Rune()}

	case *tcell.EventMouse:
		if ev.Buttons() == 512 {
			return ui.ScrollEvent{Direction: ui.ScrollUp}
		} else if ev.Buttons() == 256 {
			return ui.ScrollEvent{Direction: ui.ScrollDown}
		}
		x, y := ev.Position()
		return ui.MouseEvent{
			Position:       ui.Position{X: x, Y: y},
			Button:         ui.Button(ev.Buttons()),
			ButtonModifier: ui.ButtonModifier(ev.Modifiers()),
			Time:           ev.When(),
		}

	default:
		log.Printf("### unhandled tcell event: %#v", ev)
		return nil
	}
}

func (r *tcellDevice) Text(runes []rune, x, y int, style device.Style) {
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

func (r *tcellDevice) Show() {
	r.screen.Show()
}

func (r *tcellDevice) Stop() {
	r.screen.Fini()
}
