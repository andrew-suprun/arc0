package main

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

var defStyle = tcell.StyleDefault.Background(tcell.ColorReset).Foreground(tcell.ColorReset)

var events = make(chan struct{})

func main() {
	s, e := tcell.NewScreen()
	if e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	if e := s.Init(); e != nil {
		fmt.Fprintf(os.Stderr, "%v\n", e)
		os.Exit(1)
	}
	s.SetStyle(defStyle)

	go func() {
	outer:
		for {
			ev := s.PollEvent()
			events <- struct{}{}
			switch ev := ev.(type) {
			case *tcell.EventKey:
				if ev.Name() == "Ctrl+C" {
					close(events)
					break outer
				}
			}
		}
	}()

	for range events {
		render(s)
	}
	s.Fini()
}

func render(s tcell.Screen) {
	x, y := 0, 0
	for i := 0; i < 256; i++ {
		style := tcell.StyleDefault.Background(tcell.PaletteColor(i))
		emitStr(s, x, y, style, fmt.Sprintf(" %3d ", i))
		if i%6 == 3 {
			x = 0
			y += 1
		} else {
			x += 5
		}
	}
	s.Show()
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		s.SetContent(x, y, c, nil, style)
		x += 1
	}
}
