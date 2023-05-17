package tcell

import (
	"arch/device"
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

func (r *tcellDevice) PollEvent() device.Event {
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
		return device.ResizeEvent{Width: w, Height: h}

	case *tcell.EventKey:
		log.Printf("key: name=%v rune='%v' mod=%v", ev.Name(), ev.Rune(), ev.Modifiers())
		return device.KeyEvent{Name: ev.Name(), Rune: ev.Rune()}

	case *tcell.EventMouse:
		if ev.Buttons() == 512 {
			return device.ScrollEvent{Direction: device.ScrollUp}
		} else if ev.Buttons() == 256 {
			return device.ScrollEvent{Direction: device.ScrollDown}
		}
		x, y := ev.Position()
		return device.MouseEvent{
			X:              x,
			Y:              y,
			Button:         device.Button(ev.Buttons()),
			ButtonModifier: device.ButtonModifier(ev.Modifiers()),
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

func (r *tcellDevice) Render() {
	r.screen.Show()
}

func (r *tcellDevice) Stop() {
	r.screen.Fini()
}
