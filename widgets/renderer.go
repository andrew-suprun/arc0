package widgets

import (
	m "arch/model"
)

type Cell struct {
	Rune  rune
	Style Style
}

type MouseTargetArea struct {
	Command  any
	Position Position
	Size     Size
}

type ScrollArea struct {
	Command  any
	Position Position
	Size     Size
}

type Screen struct {
	Cells        [][]Cell
	MouseTargets []MouseTargetArea
	ScrollAreas  []ScrollArea
	Style        Style
}

func NewScreen(size m.ScreenSize) *Screen {
	screen := &Screen{
		Cells: make([][]Cell, size.Height),
	}
	for y := 0; y < len(screen.Cells); y++ {
		screen.Cells[y] = make([]Cell, size.Width)
	}
	return screen
}

type Renderer interface {
	Push(*Screen)
	Quit()
}
