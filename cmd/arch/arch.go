package main

import (
	"log"
	"os"
	"scanner/app"
)

func main() {
	os.Remove("debug.log")
	f, err := os.OpenFile("debug.log", os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	log.SetOutput(f)

	app.Run()
}
