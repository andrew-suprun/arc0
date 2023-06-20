package tcell

import (
	"arch/model"
	"arch/widgets"
	"log"

	"github.com/gdamore/tcell/v2"
)

type tcellRenderer struct {
	events           model.EventChan
	screen           tcell.Screen
	lastMouseEvent   *tcell.EventMouse
	mouseTargetAreas []mouseTargetArea
	scrollAreas      []scrollArea
	style            widgets.Style
}

type mouseTargetArea struct {
	model.MouseTarget
	widgets.Position
	widgets.Size
}

type scrollArea struct {
	model.Scroll
	widgets.Position
	widgets.Size
}

var defaultStyle = widgets.Style{FG: 231, BG: 17}

func NewRenderer(events model.EventChan) (*tcellRenderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()

	device := &tcellRenderer{
		events:         events,
		screen:         screen,
		lastMouseEvent: tcell.NewEventMouse(0, 0, 0, 0),
		style:          defaultStyle,
	}

	go func() {
		for {
			device.handleEvent()
		}
	}()

	return device, nil
}

func (device *tcellRenderer) handleEvent() {
	event := device.screen.PollEvent()
	for {
		if ev, mouseEvent := event.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
			break
		}
		event = device.screen.PollEvent()
	}

	if event == nil {
		return
	}
	switch tcellEvent := event.(type) {
	case *tcell.EventResize:
		device.screen.Sync()
		w, h := tcellEvent.Size()
		device.events <- model.ScreenSize{Width: w, Height: h}

	case *tcell.EventKey:
		device.handleKeyEvent(tcellEvent)

	case *tcell.EventMouse:
		device.handleMouseEvent(tcellEvent)

	default:
		log.Panicf("### unhandled tcell event: %#v", tcellEvent)
	}
}

func (d *tcellRenderer) AddMouseTarget(event model.MouseTarget, pos widgets.Position, size widgets.Size) {
	d.mouseTargetAreas = append(d.mouseTargetAreas, mouseTargetArea{MouseTarget: event, Position: pos, Size: size})
}
func (d *tcellRenderer) AddScrollArea(event model.Scroll, pos widgets.Position, size widgets.Size) {
	d.scrollAreas = append(d.scrollAreas, scrollArea{Scroll: event, Position: pos, Size: size})
}
func (d *tcellRenderer) SetStyle(style widgets.Style) {
	d.style = style
}

func (d *tcellRenderer) CurrentStyle() widgets.Style {
	return d.style
}

func (device *tcellRenderer) handleKeyEvent(key *tcell.EventKey) {
	log.Printf("### key: %q  %v  %c", key.Name(), key.Modifiers(), key.Rune())
	switch key.Name() {
	case "Ctrl+C":
		device.events <- model.Quit{}

	case "Enter":
		device.events <- model.Enter{}

	case "Esc":
		device.events <- model.Esc{}

	case "Ctrl+R":
		device.events <- model.RevealInFinder{}

	case "Home":
		device.events <- model.SelectFirst{}

	case "End":
		device.events <- model.SelectLast{}

	case "PgUp":
		device.events <- model.PgUp{}

	case "PgDn":
		device.events <- model.PgDn{}

	case "Up":
		device.events <- model.RenameSelection{Lines: -1}

	case "Down":
		device.events <- model.RenameSelection{Lines: 1}

	case "Ctrl+O":
		device.events <- model.KeepOne{}

	case "Ctrl+A":
		device.events <- model.KeepAll{}

	case "Backspace2": // Ctrl+Delete
		device.events <- model.Delete{}
	}
}

func (d *tcellRenderer) handleMouseEvent(event *tcell.EventMouse) {
	x, y := event.Position()

	if event.Buttons() == 256 || event.Buttons() == 512 {
		for _, target := range d.scrollAreas {
			if target.Position.X <= x && target.Position.X+target.Size.Width > x &&
				target.Position.Y <= y && target.Position.Y+target.Size.Height > y {

				if event.Buttons() == 512 {
					target.Scroll.Lines = 1
				} else {
					target.Scroll.Lines = -1
				}
				d.events <- target.Scroll
				return
			}
		}
	}

	for _, target := range d.mouseTargetAreas {
		if target.Position.X <= x && target.Position.X+target.Size.Width > x &&
			target.Position.Y <= y && target.Position.Y+target.Size.Height > y {

			d.events <- target.MouseTarget
			return
		}
	}
}

func (d *tcellRenderer) Text(runes []rune, pos widgets.Position) {
	for i, rune := range runes {
		style := tcell.StyleDefault.
			Foreground(tcell.PaletteColor(int(d.style.FG))).
			Background(tcell.PaletteColor(int(d.style.BG))).
			Bold(d.style.Flags&widgets.Bold == widgets.Bold).
			Italic(d.style.Flags&widgets.Italic == widgets.Italic).
			Reverse(d.style.Flags&widgets.Reverse == widgets.Reverse)

		d.screen.SetContent(pos.X+i, pos.Y, rune, nil, style)
	}
}

func (d *tcellRenderer) Show() {
	d.screen.Show()
}

func (d *tcellRenderer) Reset() {
	d.scrollAreas = d.scrollAreas[:0]
	d.mouseTargetAreas = d.mouseTargetAreas[:0]
}

func (d *tcellRenderer) Stop() {
	d.screen.Fini()
}
