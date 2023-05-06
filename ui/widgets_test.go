package ui

import (
	"testing"
)

func TestCalcSizes(t *testing.T) {
	for w := 0; w <= 80; w++ {
		widths := calcSizes(w, []Constraint[int]{
			{14, 0},
			{15, 2},
			{16, 3},
			{8, 0},
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
	for w := 0; w <= 80; w++ {
		row := Row(
			Text("foofoofoofoofoo"),
			FlexText("barbarbarbarbar", 2),
			FlexText("bazbazbazbazbaz", 3),
			Text("quuzquuz"),
		)
		r := &TestRenderer{}
		row.Render(r, 0, 0, W(w), 1, StyleDefault)
		if r.width != w {
			t.Fail()
		}
	}
}

type TestRenderer struct {
	width int
}

func (*TestRenderer) PollEvent() any {
	select {}
}

func (r *TestRenderer) Text(runes []rune, x X, y Y, style Style) {
	r.width += len(runes)
}

func (*TestRenderer) MouseTarget(command any, x X, y Y)  {}
func (*TestRenderer) ScrollTarget(command any, x X, y Y) {}
func (*TestRenderer) Show()                              {}
func (*TestRenderer) Sync()                              {}
func (*TestRenderer) Exit()                              {}
