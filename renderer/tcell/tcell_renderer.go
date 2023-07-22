package tcell

import (
	"arch/lifecycle"
	m "arch/model"
	"arch/stream"
	w "arch/widgets"
	"log"

	"github.com/gdamore/tcell/v2"
)

type tcellRenderer struct {
	lc               *lifecycle.Lifecycle
	controllerEvents *stream.Stream[m.Event]

	commands         *stream.Stream[inEvent]
	screen           tcell.Screen
	mouseTargetAreas []w.MouseTargetArea
	scrollAreas      []w.ScrollArea
	sync             bool
}

type inEvent interface {
	incoming()
}

type screenCommand struct {
	*w.Screen
}

func (screenCommand) incoming() {}

type quitCommand struct{}

func (quitCommand) incoming() {}

type tcellEvent struct {
	tcell.Event
}

func (tcellEvent) incoming() {}

func NewRenderer(lc *lifecycle.Lifecycle, controllerEvents *stream.Stream[m.Event]) (w.Renderer, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()

	renderer := &tcellRenderer{
		lc:               lc,
		controllerEvents: controllerEvents,
		screen:           screen,
		commands:         stream.NewStream[inEvent]("tcell"),
	}
	go renderer.handleEvents()
	go renderer.handleTcellEvents()

	return renderer, nil
}

func (r *tcellRenderer) Push(screen *w.Screen) {
	r.commands.Push(screenCommand{screen})
}

func (r *tcellRenderer) Quit() {
	r.commands.Push(quitCommand{})
}

func (r *tcellRenderer) handleEvents() {
	r.lc.Started()
	defer r.lc.Done()

	for {
		lastScreenCommandIdx := -1
		commands := r.commands.Pull()
		for idx, cmd := range commands {
			if _, ok := cmd.(screenCommand); ok {
				lastScreenCommandIdx = idx
			}
		}
		for idx, cmd := range commands {
			switch cmd := cmd.(type) {
			case screenCommand:
				if idx == lastScreenCommandIdx {
					r.renderScreen(cmd.Screen)
				}

			case quitCommand:
				r.screen.Fini()
				return

			case tcellEvent:
				r.handleTcellEvent(cmd.Event)

			default:
				log.Panicf("TCELL: UNHANDLED COMMAND: %T", cmd)
			}
		}
	}
}

func (r *tcellRenderer) handleTcellEvent(event tcell.Event) bool {
	switch event := event.(type) {
	case *tcell.EventResize:
		r.sync = true
		x, y := event.Size()
		r.controllerEvents.Push(m.ScreenSize{Width: x, Height: y})

	case *tcell.EventMouse:
		r.handleMouseEvent(event)

	case *tcell.EventKey:
		r.handleKeyEvent(event)

	default:
		log.Panicf("### unhandled renderer event: %T", event)
	}
	return true
}

func (r *tcellRenderer) renderScreen(screen *w.Screen) {
	r.mouseTargetAreas = make([]w.MouseTargetArea, len(screen.MouseTargets))
	copy(r.mouseTargetAreas, screen.MouseTargets)

	r.scrollAreas = make([]w.ScrollArea, len(screen.ScrollAreas))
	copy(r.scrollAreas, screen.ScrollAreas)

	for y := range screen.Cells {
		for x, cell := range screen.Cells[y] {
			style := tcell.StyleDefault.
				Foreground(tcell.PaletteColor(int(cell.Style.FG))).
				Background(tcell.PaletteColor(int(cell.Style.BG))).
				Bold(cell.Style.Flags&w.Bold == w.Bold).
				Italic(cell.Style.Flags&w.Italic == w.Italic).
				Reverse(cell.Style.Flags&w.Reverse == w.Reverse)

			r.screen.SetContent(x, y, cell.Rune, nil, style)
		}
	}
	if r.sync {
		r.screen.Sync()
		r.sync = false
	} else {
		r.screen.Show()
	}
}

func (r *tcellRenderer) handleTcellEvents() {
	for {
		event := r.screen.PollEvent()
		for {
			if ev, mouseEvent := event.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
				break
			}
			event = r.screen.PollEvent()
		}

		if event != nil {
			r.commands.Push(tcellEvent{event})
		}
	}
}

func (device *tcellRenderer) handleKeyEvent(key *tcell.EventKey) {
	log.Printf("### key: %q", key.Name())
	switch key.Name() {
	case "Ctrl+C":
		device.controllerEvents.Push(m.Quit{})

	case "Enter":
		device.controllerEvents.Push(m.Open{})

	// case "Esc":

	case "Ctrl+R":
		device.controllerEvents.Push(m.RevealInFinder{})

	case "Home":
		device.controllerEvents.Push(m.SelectFirst{})

	case "End":
		device.controllerEvents.Push(m.SelectLast{})

	case "PgUp":
		device.controllerEvents.Push(m.PgUp{})

	case "PgDn":
		device.controllerEvents.Push(m.PgDn{})

	case "Up":
		device.controllerEvents.Push(m.MoveSelection{Lines: -1})

	case "Down":
		device.controllerEvents.Push(m.MoveSelection{Lines: 1})

	case "Left":
		device.controllerEvents.Push(m.Exit{})

	case "Right":
		device.controllerEvents.Push(m.Enter{})

	case "Ctrl+K":
		device.controllerEvents.Push(m.KeepOne{})

	case "Ctrl+A":
		device.controllerEvents.Push(m.KeepAll{})

	case "Tab":
		device.controllerEvents.Push(m.Tab{})

	case "Backspace2": // Ctrl+Delete
		device.controllerEvents.Push(m.Delete{})
	case "F12":
		device.controllerEvents.Push(m.Debug{})
	}
}

func (d *tcellRenderer) handleMouseEvent(event *tcell.EventMouse) {
	x, y := event.Position()

	if event.Buttons() == 256 || event.Buttons() == 512 {
		for _, target := range d.scrollAreas {
			if target.Position.X <= x && target.Position.X+target.Size.Width > x &&
				target.Position.Y <= y && target.Position.Y+target.Size.Height > y {

				if event.Buttons() == 512 {
					d.controllerEvents.Push(m.Scroll{Command: target.Command, Lines: 1})
				} else {
					d.controllerEvents.Push(m.Scroll{Command: target.Command, Lines: -1})
				}
				return
			}
		}
	}

	for _, target := range d.mouseTargetAreas {
		if target.Position.X <= x && target.Position.X+target.Size.Width > x &&
			target.Position.Y <= y && target.Position.Y+target.Size.Height > y {

			d.controllerEvents.Push(m.MouseTarget{Command: target.Command})
			return
		}
	}
}
