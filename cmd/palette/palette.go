package main

import (
	"fmt"
	"log"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	for i := 0; i < 256; i++ {
		style := lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprint(i)))
		log.Print(style.Render(fmt.Sprintf("   %3v   ", i)))
		if i%6 == 3 {
			log.Println()
		}
	}
}
