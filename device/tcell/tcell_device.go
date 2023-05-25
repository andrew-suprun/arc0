package tcell

import (
	"arch/device"
	"arch/model"
	"log"
	"os/exec"

	"github.com/gdamore/tcell/v2"
)

type tcellDevice struct {
	screen           tcell.Screen
	lastMouseEvent   *tcell.EventMouse
	mouseTargetAreas []mouseTargetArea
	scrollAreas      []scrollArea
	style            device.Style
}

type mouseTargetArea struct {
	Command any
	Pos     device.Position
	Size    device.Size
}

type scrollArea struct {
	Command any
	Pos     device.Position
	Size    device.Size
}

var defaultStyle = device.Style{FG: 231, BG: 17}

func NewDevice(events model.EventChan) (*tcellDevice, error) {
	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, err
	}
	if err := screen.Init(); err != nil {
		return nil, err
	}
	screen.EnableMouse()

	device := &tcellDevice{
		screen:         screen,
		lastMouseEvent: tcell.NewEventMouse(0, 0, 0, 0),
		style:          defaultStyle,
	}

	go func() {
		for {
			event := device.screen.PollEvent()
			for {
				if ev, mouseEvent := event.(*tcell.EventMouse); !mouseEvent || ev.Buttons() != 0 {
					break
				}
				event = device.screen.PollEvent()
			}
			events <- eventHandler{device: device, event: event}
		}
	}()

	return device, nil
}

func (d *tcellDevice) AddMouseTarget(cmd any, pos device.Position, size device.Size) {
	d.mouseTargetAreas = append(d.mouseTargetAreas, mouseTargetArea{Command: cmd, Pos: pos, Size: size})
}
func (d *tcellDevice) AddScrollArea(cmd any, pos device.Position, size device.Size) {
	d.scrollAreas = append(d.scrollAreas, scrollArea{Command: cmd, Pos: pos, Size: size})
}
func (d *tcellDevice) SetStyle(style device.Style) {
	d.style = style
}

func (d *tcellDevice) CurrentStyle() device.Style {
	return d.style
}

type eventHandler struct {
	device *tcellDevice
	event  tcell.Event
}

func (e eventHandler) HandleEvent(m *model.Model) {
	if e.event == nil {
		return
	}
	switch ev := e.event.(type) {
	case *tcell.EventResize:
		e.device.screen.Sync()
		w, h := ev.Size()
		m.ScreenSize = model.Size{Width: w, Height: h}

	case *tcell.EventKey:
		e.device.handleKeyEvent(m, ev)
		makeSelectedVisible(m)

	case *tcell.EventMouse:
		if ev.Buttons() == 512 {
			m.CurerntFolder().LineOffset++
		} else if ev.Buttons() == 256 {
			m.CurerntFolder().LineOffset--
		} else {
			e.device.handleMouseEvent(m, ev)
		}

	default:
		log.Panicf("### unhandled tcell event: %#v", ev)
	}
}

func (d *tcellDevice) handleKeyEvent(m *model.Model, key *tcell.EventKey) {
	if key.Name() == "Ctrl+C" {
		m.Quit = true
	}

	loc := m.CurerntFolder()
	switch key.Name() {
	case "Enter":
		d.enter(m)

	case "Esc":
		if len(m.Breadcrumbs) > 1 {
			m.Breadcrumbs = m.Breadcrumbs[:len(m.Breadcrumbs)-1]
			m.Sort()
		}

	case "Rune[R]", "Rune[r]":
		if loc != nil {
			exec.Command("open", "-R", loc.Selected.AbsName()).Start()
		}

	case "Home":
		if len(loc.File.Files) > 0 {
			loc.Selected = loc.File.Files[0]
		}

	case "End":
		if len(loc.File.Files) > 0 {
			loc.Selected = loc.File.Files[len(loc.File.Files)-1]
		}

	case "PgUp":
		loc.LineOffset -= m.FileTreeLines
		if loc.LineOffset < 0 {
			loc.LineOffset = 0
		}
		idxSelected := 0
		foundSelected := false
		for i := 0; i < len(loc.File.Files); i++ {
			if loc.File.Files[i] == loc.Selected {
				idxSelected = i
				foundSelected = true
				break
			}
		}
		if foundSelected {
			idxSelected -= m.FileTreeLines
			if idxSelected < 0 {
				idxSelected = 0
			}
			loc.Selected = loc.File.Files[idxSelected]
		}

	case "PgDn":
		loc.LineOffset += m.FileTreeLines
		if loc.LineOffset > len(loc.File.Files)-m.FileTreeLines {
			loc.LineOffset = len(loc.File.Files) - m.FileTreeLines
		}
		idxSelected := 0
		foundSelected := false
		for i := 0; i < len(loc.File.Files); i++ {
			if loc.File.Files[i] == loc.Selected {
				idxSelected = i
				foundSelected = true
				break
			}
		}
		if foundSelected {
			idxSelected += m.FileTreeLines
			if idxSelected > len(loc.File.Files)-1 {
				idxSelected = len(loc.File.Files) - 1
			}
			loc.Selected = loc.File.Files[idxSelected]
		}

	case "Up":
		loc := m.CurerntFolder()
		if loc.Selected != nil {
			for i, file := range loc.File.Files {
				if file == loc.Selected && i > 0 {
					loc.Selected = loc.File.Files[i-1]
					break
				}
			}
		} else {
			loc.Selected = loc.File.Files[len(loc.File.Files)-1]
		}

	case "Down":
		loc := m.CurerntFolder()
		if loc.Selected != nil {
			for i, file := range loc.File.Files {
				if file == loc.Selected && i+1 < len(loc.File.Files) {
					loc.Selected = loc.File.Files[i+1]
					break
				}
			}
		} else {
			loc.Selected = loc.File.Files[0]
		}
	}
}

func (d *tcellDevice) handleMouseEvent(m *model.Model, event *tcell.EventMouse) {
	x, y := event.Position()
	for _, target := range d.mouseTargetAreas {
		if target.Pos.X <= x && target.Pos.X+target.Size.Width > x &&
			target.Pos.Y <= y && target.Pos.Y+target.Size.Height > y {

			switch cmd := target.Command.(type) {
			case model.SelectFolder:
				for i, loc := range m.Breadcrumbs {
					if loc.File == cmd && i < len(m.Breadcrumbs) {
						m.Breadcrumbs = m.Breadcrumbs[:i+1]
						m.Sort()
						return
					}
				}
			case model.SelectFile:
				m.CurerntFolder().Selected = cmd
				if event.When().Sub(d.lastMouseEvent.When()).Seconds() < 0.5 {
					d.enter(m)
				}
				d.lastMouseEvent = event
			case model.SortColumn:
				if cmd == m.SortColumn {
					m.SortAscending[m.SortColumn] = !m.SortAscending[m.SortColumn]
				} else {
					m.SortColumn = cmd
				}
				m.Sort()
			}
		}
	}
}

func (d *tcellDevice) enter(m *model.Model) {
	selected := m.CurerntFolder().Selected
	if selected == nil {
		return
	}
	if selected.Kind == model.FileFolder {
		m.Breadcrumbs = append(m.Breadcrumbs, model.Folder{File: selected})
		m.Sort()
	} else {
		exec.Command("open", selected.AbsName()).Start()
	}
}

func makeSelectedVisible(m *model.Model) {
	folder := m.CurerntFolder()
	if folder == nil || folder.Selected == nil {
		return
	}
	idx := -1
	for i := range folder.File.Files {
		if folder.Selected == folder.File.Files[i] {
			idx = i
			break
		}
	}
	if idx >= 0 {
		if folder.LineOffset > idx {
			folder.LineOffset = idx
		}
		if folder.LineOffset < idx+1-m.FileTreeLines {
			folder.LineOffset = idx + 1 - m.FileTreeLines
		}
	}
}

func (d *tcellDevice) Text(runes []rune, pos device.Position) {
	for i, rune := range runes {
		style := tcell.StyleDefault.
			Foreground(tcell.PaletteColor(int(d.style.FG))).
			Background(tcell.PaletteColor(int(d.style.BG))).
			Bold(d.style.Flags&device.Bold == device.Bold).
			Italic(d.style.Flags&device.Italic == device.Italic).
			Reverse(d.style.Flags&device.Reverse == device.Reverse)

		d.screen.SetContent(pos.X+i, pos.Y, rune, nil, style)
	}
}

func (d *tcellDevice) Show() {
	d.screen.Show()
}

func (d *tcellDevice) Reset() {
	d.scrollAreas = d.scrollAreas[:0]
	d.mouseTargetAreas = d.mouseTargetAreas[:0]
}

func (d *tcellDevice) Stop() {
	d.screen.Fini()
}
