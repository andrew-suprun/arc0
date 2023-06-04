package widgets

import (
	"testing"
)

func TestCalcSizes(t *testing.T) {
	for w := 0; w <= 80; w++ {
		widths := calcSizes(w, []int{14, 15, 16, 8}, []int{0, 2, 3, 0})
		total := 0
		for _, width := range widths {
			total += width
		}
		if total != w {
			t.Error("Expected", w, "got", total)
		}
	}
}
