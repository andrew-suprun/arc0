package ui

import (
	"log"
)

type LogRenderer struct{}

func (LogRenderer) PollEvent() any {
	select {}
}

func (LogRenderer) Write(runes []rune, x X, y Y, style Style) {
	log.Printf("Renderer: Write('%s', X(%d), Y(%d), Style(%v)", string(runes), x, y, style)
}

func (LogRenderer) Show() {
	log.Println("Renderer: Show()")
}

func (LogRenderer) Sync() {
	log.Println("Renderer: Sync()")
}

func (LogRenderer) Exit() {
	log.Println("Renderer: Exit()")
}
