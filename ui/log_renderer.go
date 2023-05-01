package ui

import (
	"log"
)

type LogRenderer struct{}

func (_ LogRenderer) PollEvent() any {
	select {}
}

func (_ LogRenderer) Write(runes []rune, x X, y Y, attributes *Attributes) {
	log.Printf("Renderer: Write('%s', X(%d), Y(%d), Attributes(%v)", string(runes), x, y, attributes)
}

func (_ LogRenderer) Show() {
	log.Println("Renderer: Show()")
}

func (_ LogRenderer) Sync() {
	log.Println("Renderer: Sync()")
}

func (_ LogRenderer) Exit() {
	log.Println("Renderer: Exit()")
}
