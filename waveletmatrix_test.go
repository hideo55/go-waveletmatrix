package waveletmatrix

import (
	"testing"
)

func TestBuildAndAccess(t *testing.T) {
	builder := NewWMBuilder()
	src := []uint64{5, 1, 0, 4, 2, 2, 0, 3}
	wm, _ := builder.Build(src)
	if wm.Size() != uint64(len(src)) {
		t.Error("Exprected", len(src), "Got", wm.Size())
	}
	for i := 0; i < len(src); i++ {
		v, found := wm.Lookup(uint64(i))
		if !found {
			t.Error("Not Found:", i)
		}
		if v != src[i] {
			t.Error("Expected", src[i], "Got", v)
		}
	}
	if r, _ := wm.Rank(uint64(3), uint64(6)); r != uint64(0) {
		t.Error("Expected", 0, "Got", r)
	}
	if r, _ := wm.Rank(uint64(0), uint64(6)); r != uint64(1) {
		t.Error("Expected", 1, "Got", r)
	}
	if r, _ := wm.Rank(uint64(0), uint64(7)); r != uint64(2) {
		t.Error("Expected", 2, "Got", r)
	}
	if r, _ := wm.Rank(uint64(2), uint64(6)); r != uint64(2) {
		t.Error("Expected", 2, "Got", r)
	}
	if r, _ := wm.Rank(uint64(5), uint64(6)); r != uint64(1) {
		t.Error("Expected", 1, "Got", r)
	}
	if _, found := wm.Rank(uint64(1), uint64(10)); found {
		t.Error("Expected", false, "Got", found)
	}
	if pos, _ := wm.Select(uint64(2), uint64(1)); pos != uint64(4) {
		t.Error("Expected", 2, "Got", pos)

	}
}
