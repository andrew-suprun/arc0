package device

import "time"

type Device interface {
	PollEvent() Event
	Text(runes []rune, x, y int, style Style)
	Render()
}

type Event interface {
	event()
}

type Style struct {
	FG, BG byte
	Flags  Flags
}

type Flags byte

const (
	Bold    Flags = 1
	Italic  Flags = 2
	Reverse Flags = 4
)

type MouseEvent struct {
	X, Y int
	Button
	ButtonModifier
	time.Time
}

func (e MouseEvent) event() {}

type Button int

const (
	LeftButton  Button = 1
	RightButton Button = 2
)

type ButtonModifier int

const (
	Shift   ButtonModifier = 1
	Control ButtonModifier = 2
	Option  ButtonModifier = 4
)

type ScrollEvent struct {
	Direction
}

func (e ScrollEvent) event() {}

type Direction int

const (
	ScrollUp   Direction = 1
	ScrollDown Direction = 2
)

type KeyEvent struct {
	Name string
	Rune rune
}

func (e KeyEvent) event() {}

type ResizeEvent struct {
	Width, Height int
}

func (e ResizeEvent) event() {}
