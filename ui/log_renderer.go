package ui

import (
	"log"
)

type LogRenderer struct{}

func (LogRenderer) PollEvent() any {
	select {}
}

func (LogRenderer) Write(runes []rune, x X, y Y, attributes *Attributes) {
	log.Printf("Renderer: Write('%s', X(%d), Y(%d), Attributes(%v)", string(runes), x, y, attributes)
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
