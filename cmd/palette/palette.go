package main

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

func main() {
	for i := 0; i < 256; i++ {
		style := lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprint(i))).Foreground(lipgloss.Color("231"))
		if i >= 16 && i <= 21 {
			style = lipgloss.NewStyle().Background(lipgloss.Color(fmt.Sprint(i))).Foreground(lipgloss.Color("118")).Bold(true).Italic(true)
		}
		fmt.Print(style.Render(fmt.Sprintf("   %3v   ", i)))
		if i%6 == 3 {
			fmt.Println()
		}
	}
}
