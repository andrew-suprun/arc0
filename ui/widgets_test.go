package ui

import (
	"log"
	"testing"
)

func TestCalcWidths(t *testing.T) {
	for w := 0; w <= 60; w++ {
		widths := calcSizes(w, []Widget{
			Text("foofoofoofoofoo"),
			FlexText("barbarbarbarbar", 2),
			FlexText("bazbazbazbazbaz", 3),
			Text("quuzquuz"),
		})
		total := 0
		for _, width := range widths {
			total += width
		}
		if total != w {
			t.Error("Expected", w, "got", total)
		}
	}
}

func TestRow(t *testing.T) {
	for w := 0; w <= 60; w++ {
		row := Row(
			Text("foofoofoofoofoo"),
			FlexText("barbarbarbarbar", 2),
			FlexText("bazbazbazbazbaz", 3),
			Text("quuzquuz"),
		)

		segments := row.Render(Position{}, Size{Width, w}, DefaultAttributes().MouseTarget("FOO"))
		total := 0
		for _, segment := range segments {
			total += len(segment.Runes)
		}
		if total != w {
			log.Println("----")
			for _, segment := range segments {
				log.Printf("%v: '%s'", segment.Position, string(segment.Runes))
			}
			t.Error("Expected", w, "got", total)
		}
	}
}
