package ui

import (
	"testing"
)

func TestCalcWidths(t *testing.T) {
	for w := 0; w <= 60; w++ {
		widths := calcSizes(w, []Widget{
			Text("foofoofoofoofoo", 0),
			Text("barbarbarbarbar", 2),
			Text("bazbazbazbazbaz", 3),
			Text("quuzquuz", 0),
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
			Text("foofoofoofoofoo", 0),
			Text("barbarbarbarbar", 2),
			Text("bazbazbazbazbaz", 3),
			Text("quuzquuz", 0),
		)

		segments := row.Render(Position{}, Size{Width, w}, DefaultAttributes().MouseTarget("FOO"))
		total := 0
		for _, segment := range segments {
			total += len(segment.Runes)
		}
		if total != w {
			t.Error("Expected", w, "got", total)
		}
	}
}
