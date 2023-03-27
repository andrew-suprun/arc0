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
		style := tcell.StyleDefault.Background(tcell.PaletteColor(i)).Bold(true)
		j := i % 36
		if j >= 16 && j < 34 || i < 10 || i == 12 || i == 13 {
			style = style.Foreground(tcell.PaletteColor(231))
		} else {
			style = style.Foreground(tcell.PaletteColor(0))
		}

		emitStr(s, x, y, style, fmt.Sprintf("   %3d   ", i))
		if i >= 16 && i%6 == 3 || i < 16 && i%4 == 3 {
			x = 0
			y++
		} else {
			x += 10
		}
		if i%36 == 15 {
			y++
		}
	}
	s.Show()
}

func emitStr(s tcell.Screen, x, y int, style tcell.Style, str string) {
	for _, c := range str {
		s.SetContent(x, y, c, nil, style)
		x++
	}
}
